package config

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// divider is what comes between a key's words and its tail.
const divider = " :: "

// What an indent is made of. An indent of both is mixed.
const (
	spaces = "spaces"
	tabs   = "tabs"
	mixed  = "both"
)

// noKey is why a line is refused when nothing it can read as a key comes
// before its divider. Spaces of another kind can do that.
const noKey = `there is no key before the "::"`

// reading is one file being read into a copy of the value. It keeps the
// file's name and what the file may set, every line refused so far, and,
// while a block is open, that block.
type reading struct {
	name    string
	value   reflect.Value
	keys    []key
	seen    map[string]bool
	cleared map[int]bool
	refused []error
	open    *opened
}

// opened is a block being read: the element its lines fill, the keys
// they may set, and the kind of indent the block began with.
type opened struct {
	name     string
	element  reflect.Value
	keys     []key
	seen     map[string]bool
	cleared  map[int]bool
	indented string
}

// Read fills into from raw, a file named name in its errors. It refuses
// the whole file, with every bad line reported together as
// "name:line: key: why", or fills nothing: into is untouched on an error.
//
// Defaults are whatever into held before. A key the file does not set
// leaves its field as it was. A slice key the file does set replaces the
// field's slice with the file's lines, in file order.
//
// The lines it refuses are:
//
//   - a key that is not a setting here
//   - the wrong number of words for the range
//   - a second line for a scalar key
//   - an int that is not a whole number
//   - a tail where the key takes none, or none where it takes one
//   - an indent of tabs mixed with spaces, a line that changes the kind
//     of indent inside its block, and an indented line under no block
//   - an error from a LineSetter
func (schema Schema[File]) Read(name string, raw []byte, into *File) error {
	if into == nil {
		return fmt.Errorf("%s: there is nowhere to read the file into",
			name)
	}
	work := *into
	file := &reading{name: name, value: reflect.ValueOf(&work).Elem(),
		keys: schema.keys, seen: map[string]bool{},
		cleared: map[int]bool{}}
	for at, text := range strings.Split(string(raw), "\n") {
		file.line(at+one, strings.TrimRight(text, "\r"))
	}
	if err := errors.Join(file.refused...); err != nil {
		return err
	}
	*into = work
	return nil
}

// indentKind is what an indent is made of: spaces or tabs, or mixed when
// it has both.
func indentKind(indent string) string {
	switch {
	case strings.Contains(indent, "\t") && strings.Contains(indent, " "):
		return mixed
	case strings.Contains(indent, "\t"):
		return tabs
	}
	return spaces
}

// line reads one line: a comment or a blank, a key, or a key inside the
// block above it.
func (file *reading) line(at int, text string) {
	trimmed := strings.TrimLeft(text, " \t")
	if strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "#") {
		return
	}
	indent := text[:len(text)-len(trimmed)]
	kind := indentKind(indent)
	switch {
	case kind == mixed:
		file.refuse(at, firstWord(trimmed), "it is indented with both "+
			"tabs and spaces; use one or the other")
	case indent == "":
		file.open = nil
		file.top(at, trimmed)
	case file.open == nil:
		file.refuse(at, firstWord(trimmed), "it is indented, but "+
			"under no block")
	default:
		file.inside(at, trimmed, kind)
	}
}

// firstWord is the key a line seems to have, to name it in a refusal.
// The line is not blank, so it has a word.
func firstWord(text string) string {
	return strings.Fields(text)[first]
}

// refuse keeps why a line cannot be read, by the file's name, the line
// and the key.
func (file *reading) refuse(at int, name, why string) {
	named := ": "
	if name != "" {
		named = ": " + name + ": "
	}
	file.refused = append(file.refused,
		fmt.Errorf("%s:%d%s%s", file.name, at, named, why))
}

// top reads a line at the left margin: a key of the file.
func (file *reading) top(at int, text string) {
	each, line, ok := file.parse(at, text, file.keys, "")
	if !ok {
		return
	}
	if each.shape == single && file.seen[each.name] {
		file.refuse(at, each.name, twice)
		return
	}
	file.seen[each.name] = true
	field := file.value.Field(each.field)
	if each.shape != block {
		file.set(at, each, field, file.cleared, line)
		return
	}
	file.begin(at, each, field, line)
}

// twice is why a second line of a key that is on one line at most is
// refused.
const twice = "this key is on one line at most, and it is set again"

// begin opens a block: a new element in its field, read from the
// opening line.
func (file *reading) begin(at int, each key, field reflect.Value,
	line Line) {
	if !file.cleared[each.field] {
		field.Set(reflect.Zero(field.Type()))
		file.cleared[each.field] = true
	}
	fresh := reflect.New(field.Type().Elem()).Elem()
	field.Set(reflect.Append(field, fresh))
	element := field.Index(field.Len() - one)
	file.open = &opened{name: each.name, element: element,
		keys: each.element.keys, seen: map[string]bool{},
		cleared: map[int]bool{}}
	opening := each.element.opening
	words := element.Field(opening.field)
	if err := fill(words, opening.kind, line); err != nil {
		file.refuse(at, each.name, err.Error())
	}
}

// inside reads a line indented under the open block.
func (file *reading) inside(at int, text, kind string) {
	block := file.open
	if block.indented == "" {
		block.indented = kind
	}
	if kind != block.indented {
		file.refuse(at, firstWord(text), fmt.Sprintf("it is indented "+
			"with %s, but this block is indented with %s", kind,
			block.indented))
		return
	}
	each, line, ok := file.parse(at, text, block.keys, block.name)
	if !ok {
		return
	}
	if each.shape == single && block.seen[each.name] {
		file.refuse(at, each.name, twice)
		return
	}
	block.seen[each.name] = true
	field := block.element.Field(each.field)
	file.set(at, each, field, block.cleared, line)
}

// set reads a line into a field that is not a block. A slice field is
// replaced by the file's lines at its first line.
func (file *reading) set(at int, each key, field reflect.Value,
	cleared map[int]bool, line Line) {
	var err error
	if each.shape == many {
		if !cleared[each.field] {
			field.Set(reflect.Zero(field.Type()))
			cleared[each.field] = true
		}
		elem := reflect.New(field.Type().Elem()).Elem()
		err = fill(elem, each.kind, line)
		field.Set(reflect.Append(field, elem))
	} else {
		err = fill(field, each.kind, line)
	}
	if err != nil {
		file.refuse(at, each.name, err.Error())
	}
}

// fill reads a line into a value of the kind.
func fill(into reflect.Value, how kind, line Line) error {
	switch how {
	case asSetter:
		setter := into.Addr().Interface().(LineSetter)
		return setter.SetLine(line)
	case asInt:
		got, err := strconv.Atoi(line.Words[first])
		if err != nil {
			return fmt.Errorf("%q is not a whole number",
				line.Words[first])
		}
		into.SetInt(int64(got))
	case asWords:
		words := reflect.ValueOf(slices.Clone(line.Words))
		into.Set(words.Convert(into.Type()))
	default:
		into.SetString(strings.Join(line.Words, " "))
	}
	return nil
}

// parse reads one line as the key among may that its first word names,
// and checks its words and its tail against that key. The inside is the
// block the line is in, or nothing.
func (file *reading) parse(at int, text string, may []key,
	inside string) (key, Line, bool) {
	head, tail, tailed := strings.Cut(text, divider)
	tail = strings.TrimSpace(tail)
	words := strings.Fields(head)
	if len(words) == none {
		file.refuse(at, "", noKey)
		return key{}, Line{}, false
	}
	name := words[first]
	words = words[one:]
	if len(words) > none && words[first] == "=" {
		words = words[one:]
	}
	index := slices.IndexFunc(may, func(each key) bool {
		return each.name == name
	})
	if index < none {
		file.refuse(at, name, unknown(may, inside))
		return key{}, Line{}, false
	}
	each := may[index]
	words, tailed = emptyTail(each, words, tailed)
	if why := each.takes(len(words), tailed, tail); why != "" {
		file.refuse(at, name, why)
		return key{}, Line{}, false
	}
	line := Line{Words: words, At: at}
	if each.tail {
		line.Tail = tail
	}
	return each, line, true
}

// emptyTail is a line's words without a last "::", when its key takes a
// tail and the line ends in one: a tail with nothing in it. For any other
// key "::" is a word, such as an address to listen on.
func emptyTail(each key, words []string, tailed bool) ([]string, bool) {
	last := len(words) - one
	if tailed || !each.tail || last < none || words[last] != "::" {
		return words, tailed
	}
	return words[:last], true
}

// takes is why a line does not take what its key declares, or nothing
// when it does.
func (each key) takes(words int, tailed bool, tail string) string {
	switch {
	case words < each.least || each.most != noMost && words > each.most:
		return fmt.Sprintf("this key takes %s, and this line has %d",
			each.wanted(), words)
	case tailed && !each.tail:
		return fmt.Sprintf("this key takes no tail after %q",
			strings.TrimSpace(divider))
	case each.tail && tail == "":
		return fmt.Sprintf("this key takes a tail after %q",
			strings.TrimSpace(divider))
	}
	return ""
}

// wanted is the words a key takes, said: "1 word", "2 to 3 words" or "at
// least 2 words".
func (each key) wanted() string {
	switch {
	case each.most == each.least:
		return counted(each.least)
	case each.most == noMost:
		return "at least " + counted(each.least)
	}
	return fmt.Sprintf("%d to %s", each.least, counted(each.most))
}

// counted is a number of words, said.
func counted(words int) string {
	if words == one {
		return "1 word"
	}
	return fmt.Sprintf("%d words", words)
}

// unknown is why a key that is not set here is refused, with the keys
// there are.
func unknown(may []key, inside string) string {
	var names []string
	for _, each := range may {
		names = append(names, each.name)
	}
	list := strings.Join(names, ", ")
	if inside != "" {
		return fmt.Sprintf("this is not a key of %s; its keys are %s",
			inside, list)
	}
	return fmt.Sprintf("this is not a key; the keys are %s", list)
}
