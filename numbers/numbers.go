// Package numbers reads a numbers file. It lists every number a project
// uses that is not true by definition. Each number has a name, says what
// it is and says where it came from.
//
// Any measured number names its source. That is the source's key in the
// project's sources.yaml, the place in it and the text read there. A
// picked number says who picked it and when.
//
// The rules, 10-18: a measured number is pulled from its source and
// never typed from memory. Any picked number is config, in yml. The code
// names them and asks for them here. It holds none of its own.
//
// The file is read whole through go.uber.org/config in its strict mode.
// The file is refused for a key nobody knows, a key written twice, or
// an entry missing what its kind needs.
package numbers

import (
	"bytes"
	"fmt"

	"go.uber.org/config"
)

// Kind is where a number came from: measured, or picked.
type Kind string

const (
	// Measured is a number read from a published source.
	Measured Kind = "measured"
	// Picked is a number someone chose, and said why.
	Picked Kind = "picked"
)

// Source is where a measured number was read. It holds the source's key
// in sources.yaml, the place in it, the page of a paged source and the
// text found there.
type Source struct {
	key, where, text string
	page             int
}

// Key is the source's key in the project's sources.yaml.
func (from Source) Key() string { return from.key }

// Where is the place in the source: page, table, row and column.
func (from Source) Where() string { return from.where }

// Page is the page of a paged source the text is on, counted as the
// file counts them; none for a source that is not paged.
func (from Source) Page() int { return from.page }

// Text is what the source reads there, as a check would find it.
func (from Source) Text() string { return from.text }

// Number is one named number. It has a value and a unit and a line on
// what it is, and a kind. Each measured one has its source. Each picked one
// says who picked it.
type Number struct {
	name, unit, says, by string
	value                float64
	kind                 Kind
	from                 Source
}

// Name is what the code asks for the number by.
func (number Number) Name() string { return number.name }

// Value is the number.
func (number Number) Value() float64 { return number.value }

// Unit is the number's unit, as written; none for a pure number.
func (number Number) Unit() string { return number.unit }

// Says is what the number is.
func (number Number) Says() string { return number.says }

// Kind is where the number came from.
func (number Number) Kind() Kind { return number.kind }

// From is the source a measured number was read from.
func (number Number) From() Source { return number.from }

// By is who picked a picked number, and when.
func (number Number) By() string { return number.by }

// Numbers is a numbers file read: every number by its name.
type Numbers struct {
	file   string
	byName map[string]Number
	order  []string
}

// Read is the numbers a file holds. It refuses a file that is not YAML.
// It refuses a key nobody knows and a name written twice. It refuses an
// entry missing what its kind needs.
//
// It refuses a name the YAML reader may have changed. Those names are
// true, false and anything that parses as a number in any base. The
// reader turns yes into true and 010 into 8 before the code sees the
// name.
//
// Quotes do not help. The reader strips them, so '0x10' is refused as well.
// The message ends "call the number something else". Quoted names that
// are no number are taken as written, as 'yes' is.
func Read(file string, raw []byte) (Numbers, error) {
	source := config.RawSource(bytes.NewReader(raw))
	provider, err := config.NewYAML(config.Name(file), source)
	if err != nil {
		return Numbers{}, fmt.Errorf("%s: %w", file, err)
	}
	var written wire
	if err := provider.Get(config.Root).Populate(&written); err != nil {
		return Numbers{}, fmt.Errorf("%s: %w", file, err)
	}
	read := Numbers{file: file, byName: map[string]Number{}}
	for _, entry := range written.Numbers {
		number, err := entry.number()
		if err != nil {
			return Numbers{}, fmt.Errorf("%s: %w", file, err)
		}
		if _, twice := read.byName[number.name]; twice {
			return Numbers{}, fmt.Errorf("%s: %s is written twice",
				file, number.name)
		}
		read.byName[number.name] = number
		read.order = append(read.order, number.name)
	}
	return read, nil
}

// Get is the number of a name, or a refusal naming the file it is not in.
func (read Numbers) Get(name string) (Number, error) {
	number, ok := read.byName[name]
	if !ok {
		return Number{}, fmt.Errorf("%s holds no number called %s",
			read.file, name)
	}
	return number, nil
}

// All is every number in the file, in the order written.
func (read Numbers) All() []Number {
	all := make([]Number, len(read.order))
	for place, name := range read.order {
		all[place] = read.byName[name]
	}
	return all
}
