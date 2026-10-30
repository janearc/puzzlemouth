package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Setting is one key a Schema allows, as Settings returns it. It is only
// read, so its fields are plain.
type Setting struct {
	// Key is the key, as tagged.
	Key string
	// Words is the range as tagged: "1", "2-3" or "1+".
	Words string
	// Help is the help line, as tagged.
	Help string
	// Tail is whether the line takes a tail after " :: ".
	Tail bool
	// Block is whether the key opens a block.
	Block bool
	// In is, for a block, the settings its indented lines may set, in
	// field order. For any other key it is nil.
	In []Setting
}

// Line is one line of the file, handed to a field type that means more
// than its words. It is passed in and never returned, so its fields are
// plain.
type Line struct {
	// Words are the words after the key, before any tail. For the line
	// that opens a block they are the words after the block's key.
	Words []string
	// Tail is what follows " :: ", trimmed, when the key takes a tail.
	// It is empty for a key that takes none.
	Tail string
	// At is the line in the file, counted from one.
	At int
}

// LineSetter is implemented by a field's type to read a line itself.
// SetLine is called once for each line of the key, in file order. For a
// slice field it is called on a fresh element, which is then added.
//
// An error it returns is reported as name:line: key: why, with the rest
// of the file's errors.
type LineSetter interface{ SetLine(Line) error }

// Schema is what a file of type File may say. NewSchema makes it, once,
// from File's fields and their tags. The zero Schema allows no key, so
// it refuses every line of any file.
type Schema[File any] struct {
	keys     []key
	settings []Setting
}

// shape is how many lines a key may be on, and what a line opens.
type shape int

const (
	// single is a key on one line at most.
	single shape = iota + 1
	// many is a key on any number of lines, one element each.
	many
	// block is a key whose line opens a block of indented lines.
	block
)

// kind is what a line is read into.
type kind int

const (
	// asString is a string, from one word or the words joined.
	asString kind = iota + 1
	// asInt is an int, from exactly one word.
	asInt
	// asWords is a []string, the words themselves.
	asWords
	// asSetter is a type that reads its own Line.
	asSetter
)

// key is one key of a file, or of a block, as its field declares it.
type key struct {
	name  string
	field int
	// goName is the field's name, for the declaration's refusals.
	goName string
	words  string
	// least and most are the words the key takes. A most of noMost is
	// any number.
	least, most int
	tail        bool
	shape       shape
	kind        kind
	help        string
	// element is, for a block, what its lines are read into.
	element *element
}

// element is a block: the keys its indented lines set, and the field,
// if any, that its opening line is read into.
type element struct {
	keys    []key
	opening *key
}

const (
	// none is the count of nothing. It's defined.
	none = 0
	// noMost is a most that is no most: any number of words. It's
	// defined.
	noMost = -1
	// one is a count of one. It's defined.
	one = 1
	// first is the first place in a list. It's defined.
	first = 0
	// keyPart is the place of the key in a config tag. It's defined.
	keyPart = 0
	// wordsPart is the place of the range in a config tag. It's defined.
	wordsPart = 1
	// tailPart is the place of the tail in a config tag. It's defined.
	tailPart = 2
	// decimal is the base a range is written in. It's defined.
	decimal = 10
	// intBits is the size a count of words is read at. It's defined: it
	// fits an int on every machine.
	intBits = 31
)

// keyed is what a key may be.
var keyed = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// setterType is LineSetter, to ask whether a type implements it.
var setterType = reflect.TypeFor[LineSetter]()

// NewSchema is what a file of type File may say, made once from File's
// fields and their config tags. File must be a struct. It is called once
// per type, in a package-level var or at start-up, and a test calls it.
//
// The tag is config:"key,words" or config:"key,words,tail", with
// help:"one line" beside it. The words are N (exactly N), N-M (from N to
// M) or N+ (at least N). They are counted before any tail. The word
// tail means the line takes the rest after " :: ", and must have one.
//
// The type of the field says the shape of the key:
//
//   - A scalar field is a key on one line at most. A second line is
//     refused.
//   - A slice field is a key on any number of lines, each one element,
//     in file order.
//   - A slice of structs that have config-tagged fields is a block. Each
//     line of the key opens an element, and the indented lines under it
//     set that element's fields. A block is one level deep.
//
// What a line may be read into:
//
//   - string takes one word, or all the words joined by single spaces
//     when its range allows more than one.
//   - int takes exactly one word, and the word must be a whole number.
//   - []string takes the words of its line. It is a scalar: one line at
//     most. Use [][]string for a key on many lines.
//   - a type whose pointer implements LineSetter reads the Line itself.
//     It is the only type that can have a tail.
//
// Any other type is refused. A field with no config tag is not part of
// the file.
//
// The line that opens a block has words too, as in "project alpha".
// They are read into the one field of the element tagged
// opening:"help", which has no config tag and takes the same types as
// any scalar key.
//
// A block takes words, since one that takes none and takes no tail
// could say nothing. So it must have that field.
//
// NewSchema refuses every mistake in the declaration at once, each named
// by its field as "field Name: why" and joined together:
//
//   - a key that is not lowercase words joined by hyphens
//   - a key used twice, in the struct or in the same block
//   - help that is missing or is more than one line
//   - a range that cannot be met, or that is not a range: "2-1", "0"
//     with no tail, "many"
//   - a range a type cannot take: int takes exactly 1
//   - a field type it cannot fill
//   - a tail on a type that is not a LineSetter, or on a block
//   - a block inside a block
//   - a block whose opening words have nowhere to go, an opening field
//     on a type that is not a block, or two of them
//
// On any refusal the Schema is nil, so a caller that skips the error
// meets a nil pointer and not a schema that says nothing.
func NewSchema[File any]() (*Schema[File], error) {
	typ := reflect.TypeFor[File]()
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config: a file is declared as a "+
			"struct, and %s is not one", typ)
	}
	declared := &declaring{}
	got := declared.element(typ, false, nil)
	if err := errors.Join(declared.refused...); err != nil {
		return nil, err
	}
	return &Schema[File]{keys: got.keys, settings: settingsOf(got.keys)},
		nil
}

// declaring is a declaration being read, with every mistake in it so far.
type declaring struct {
	refused []error
}

// refuse keeps why a field cannot be declared as it is.
func (declared *declaring) refuse(field, why string) {
	declared.refused = append(declared.refused,
		fmt.Errorf("field %s: %s", field, why))
}

// element is the keys of a struct, and the field that opens it if it is
// a block. The parent is the key of the block it is, or nil for the file.
func (declared *declaring) element(typ reflect.Type, inside bool,
	parent *key) *element {
	got := &element{}
	seen := map[string]bool{}
	for place := range typ.NumField() {
		field := typ.Field(place)
		tagged, isKey := field.Tag.Lookup("config")
		_, isOpening := field.Tag.Lookup("opening")
		switch {
		case !isKey && !isOpening:
			continue
		case !field.IsExported():
			declared.refuse(field.Name, "it is tagged, and not "+
				"exported")
		case isKey && isOpening:
			declared.refuse(field.Name, "it has a config tag "+
				"and an opening tag; it is one or the other")
		case isOpening:
			declared.opening(got, field, place, inside)
		default:
			each := declared.key(field, place, tagged, inside)
			declared.add(got, seen, field, each)
		}
	}
	declared.settle(got, parent)
	return got
}

// add keeps a key that was declared, if it is the first of its name.
func (declared *declaring) add(got *element, seen map[string]bool,
	field reflect.StructField, each *key) {
	if each == nil {
		return
	}
	if seen[each.name] {
		declared.refuse(field.Name, fmt.Sprintf("the key %s is "+
			"used twice", each.name))
		return
	}
	seen[each.name] = true
	got.keys = append(got.keys, *each)
}

// opening keeps the field a block's opening line is read into.
func (declared *declaring) opening(got *element, field reflect.StructField,
	place int, inside bool) {
	help := field.Tag.Get("opening")
	switch {
	case !inside:
		declared.refuse(field.Name, "only a block has an opening line")
	case got.opening != nil:
		declared.refuse(field.Name, "a block has one opening field")
	case helpRefusal(help) != "":
		declared.refuse(field.Name, helpRefusal(help))
	default:
		each := &key{field: place, help: help, shape: single}
		if declared.scalar(field, each) {
			got.opening = each
		}
	}
}

// settle checks a block's opening field against its range. The parent is
// nil for the file, which is not a block.
func (declared *declaring) settle(got *element, parent *key) {
	if parent == nil {
		return
	}
	if got.opening == nil {
		declared.refuse(parent.goName, "a block's opening line has "+
			"words, and the block has no field tagged opening to "+
			"read them into")
		return
	}
	got.opening.least, got.opening.most = parent.least, parent.most
	declared.fits(parent.goName, got.opening)
}

// fits checks that a range is one the kind of the key can take.
func (declared *declaring) fits(field string, each *key) {
	if each.kind == asInt && (each.least != one || each.most != one) {
		declared.refuse(field, "an int takes exactly 1 word")
	}
}

// helpRefusal is why a help line is not one, or nothing.
func helpRefusal(help string) string {
	if strings.TrimSpace(help) == "" ||
		strings.ContainsAny(help, "\r\n") {
		return "its help is not one line"
	}
	return ""
}

// key is a field's key as its tag declares it, or nil when the tag
// cannot be read into one. Every mistake is kept.
func (declared *declaring) key(field reflect.StructField, place int,
	tagged string, inside bool) *key {
	before := len(declared.refused)
	each := &key{field: place, goName: field.Name,
		help: field.Tag.Get("help")}
	parts := strings.Split(tagged, ",")
	each.name = parts[keyPart]
	if !keyed.MatchString(each.name) {
		declared.refuse(field.Name, fmt.Sprintf("%q is not lowercase "+
			"words joined by hyphens", each.name))
	}
	if why := helpRefusal(each.help); why != "" {
		declared.refuse(field.Name, why)
	}
	declared.words(field.Name, each, parts)
	declared.shape(field, each, inside)
	declared.tail(field.Name, each)
	if len(declared.refused) > before {
		return nil
	}
	return each
}

// words reads the range and the tail of a tag's parts into the key.
func (declared *declaring) words(field string, each *key, parts []string) {
	tailed := len(parts) > tailPart
	switch {
	case len(parts) <= wordsPart:
		declared.refuse(field, "it has no range of words; the tag is "+
			"key,words")
		return
	case len(parts) > tailPart+one || tailed && parts[tailPart] != "tail":
		declared.refuse(field, "a tag is key,words or key,words,tail")
		return
	}
	each.tail = tailed
	each.words = strings.TrimSpace(parts[wordsPart])
	least, most, err := parseRange(each.words)
	if err != nil {
		declared.refuse(field, err.Error())
		return
	}
	each.least, each.most = least, most
	if most == none && !each.tail {
		declared.refuse(field, "it takes no words and no tail, so "+
			"it can say nothing")
	}
}

// parseRange reads N, N-M or N+ into its fewest and most words. A most
// of noMost is any number.
func parseRange(text string) (int, int, error) {
	not := fmt.Errorf("%q is not a range of words: N, N-M or N+", text)
	rest, open := strings.CutSuffix(text, "+")
	from, to, ranged := strings.Cut(rest, "-")
	least, err := strconv.ParseUint(from, decimal, intBits)
	if err != nil || open && ranged {
		return none, none, not
	}
	switch {
	case open:
		return int(least), noMost, nil
	case !ranged:
		return int(least), int(least), nil
	}
	most, err := strconv.ParseUint(to, decimal, intBits)
	if err != nil {
		return none, none, not
	}
	if most < least {
		return none, none, fmt.Errorf("%q takes at least %d words "+
			"and at most %d: it cannot be met", text, least, most)
	}
	return int(least), int(most), nil
}

// scalarKind is what a type is read into as a single line, or false.
func scalarKind(typ reflect.Type) (kind, bool) {
	switch {
	case reflect.PointerTo(typ).Implements(setterType):
		return asSetter, true
	case typ.Kind() == reflect.String:
		return asString, true
	case typ.Kind() == reflect.Int:
		return asInt, true
	case typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.String:
		return asWords, true
	}
	return none, false
}

// isBlock is whether a type is a struct that has tagged fields.
func isBlock(typ reflect.Type) bool {
	if typ.Kind() != reflect.Struct {
		return false
	}
	for place := range typ.NumField() {
		tags := typ.Field(place).Tag
		_, isKey := tags.Lookup("config")
		_, isOpening := tags.Lookup("opening")
		if isKey || isOpening {
			return true
		}
	}
	return false
}

// scalar sets the kind of a key that is one line, such as an opening
// field. It says whether the type of the field can be read into.
func (declared *declaring) scalar(field reflect.StructField,
	each *key) bool {
	got, ok := scalarKind(field.Type)
	if !ok {
		declared.cannotFill(field)
		return false
	}
	each.kind = got
	return true
}

// cannotFill refuses a field whose type no line can be read into.
func (declared *declaring) cannotFill(field reflect.StructField) {
	declared.refuse(field.Name, fmt.Sprintf("%s is not a type a line "+
		"can be read into", field.Type))
}

// shape sets the shape and kind of a key from the type of its field, and
// reads a block's element.
func (declared *declaring) shape(field reflect.StructField, each *key,
	inside bool) {
	typ := field.Type
	if got, ok := scalarKind(typ); ok {
		each.shape, each.kind = single, got
		return
	}
	if typ.Kind() != reflect.Slice {
		declared.cannotFill(field)
		return
	}
	if got, ok := scalarKind(typ.Elem()); ok {
		each.shape, each.kind = many, got
		return
	}
	if !isBlock(typ.Elem()) {
		declared.cannotFill(field)
		return
	}
	each.shape = block
	if inside {
		declared.refuse(field.Name, "a block inside a block; a file "+
			"is one level deep")
		return
	}
	each.element = declared.element(typ.Elem(), true, each)
}

// tail checks a key's tail and range against its kind.
func (declared *declaring) tail(field string, each *key) {
	switch {
	case each.shape == none:
		return
	case each.shape == block && each.tail:
		declared.refuse(field, "a block takes no tail")
	case each.shape != block && each.tail && each.kind != asSetter:
		declared.refuse(field, "only a type that reads its own line, "+
			"a LineSetter, can take a tail")
	case each.shape != block:
		declared.fits(field, each)
	}
}

// settingsOf is the settings the keys declare, in field order.
func settingsOf(keys []key) []Setting {
	var settings []Setting
	for _, each := range keys {
		set := Setting{Key: each.name, Words: each.words,
			Help: each.help, Tail: each.tail,
			Block: each.shape == block}
		if each.shape == block {
			set.In = settingsOf(each.element.keys)
		}
		settings = append(settings, set)
	}
	return settings
}

// Settings are the keys the file may set, in field order, for help pages
// and syntax files. A block's settings are in its In.
func (schema Schema[File]) Settings() []Setting {
	return cloneSettings(schema.settings)
}

// cloneSettings is a copy of settings that shares nothing with them.
func cloneSettings(settings []Setting) []Setting {
	var copied []Setting
	for _, each := range settings {
		each.In = cloneSettings(each.In)
		copied = append(copied, each)
	}
	return copied
}
