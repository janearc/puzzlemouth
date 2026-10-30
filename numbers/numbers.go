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
// The file is read whole by package yamlfile, which refuses a key nobody
// knows, a key written twice and a value of the wrong type, each at its
// line. Read then refuses a number missing what its kind needs.
package numbers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/janearc/puzzlemouth/yamlfile"
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
	// Key is the source's key in the project's sources.yaml.
	Key string `yaml:"key"`
	// Where is the place in the source: page, table, row and column.
	Where string `yaml:"where"`
	// Page is the page of a paged source the text is on, counted as the
	// file counts them; none for a source that is not paged.
	Page int `yaml:"page"`
	// Text is what the source reads there, as a check would find it.
	Text string `yaml:"text"`
}

// Number is one named number. It has a value and a unit and a line on
// what it is, and a kind. Each measured one has its source. Each picked one
// says who picked it.
type Number struct {
	// Name is what the code asks for the number by.
	Name string `yaml:"name"`
	// Value is the number.
	Value float64 `yaml:"value"`
	// Unit is the number's unit, as written; none for a pure number.
	Unit string `yaml:"unit"`
	// Says is what the number is.
	Says string `yaml:"says"`
	// Kind is where the number came from.
	Kind Kind `yaml:"kind"`
	// From is the source a measured number was read from. A picked
	// number has none.
	From *Source `yaml:"source"`
	// By is who picked a picked number, and when. A measured number has
	// none.
	By string `yaml:"by"`
}

// Numbers is a numbers file read: every number, in the order written.
type Numbers []Number

const (
	// fromPrefix asks ParseInt to take the base from the prefix, as 0x10
	// and 010 are written. It's defined by strconv.ParseInt: base 0.
	fromPrefix = 0
	// bits is the size a number is read at. It's defined: a float64 is
	// 64 bits.
	bits = 64
)

// Read is the numbers a file holds, or nil and why not. It refuses what
// package yamlfile refuses: a file that is not YAML, a key nobody knows,
// a key written twice, a value of the wrong type.
//
// It refuses an entry missing what its kind needs, a name written twice
// and a name that reads as a number.
//
// A name that parses as a number in any base is refused: 010 and 0x10
// are numbers, and nobody should ask for Get("010"). The message ends
// "call the number something else". A name such as yes or true is a
// word and is taken as written.
//
// Every refusal is reported together, with the file named once. A refusal
// by yamlfile is reported alone, as it stops the file being read.
func Read(file string, raw []byte) (Numbers, error) {
	var shown struct {
		Numbers []map[string]any `yaml:"numbers"`
	}
	if err := yamlfile.Read(file, raw, &shown); err != nil {
		return nil, err
	}
	var written struct {
		Numbers Numbers `yaml:"numbers"`
	}
	if err := yamlfile.Read(file, raw, &written); err != nil {
		return nil, err
	}
	var refused []error
	seen := map[string]bool{}
	for place, each := range written.Numbers {
		_, valued := shown.Numbers[place]["value"]
		refused = append(refused, each.judged(place, valued, seen))
	}
	if err := errors.Join(refused...); err != nil {
		return nil, fmt.Errorf("%s:\n  %s", file, strings.ReplaceAll(
			err.Error(), "\n", "\n  "))
	}
	return written.Numbers, nil
}

// judged is why the entry at place is refused, or nothing. Valued is
// whether the file wrote a value for it. Seen holds the names already
// read, and gets this one.
func (number Number) judged(place int, valued bool,
	seen map[string]bool) error {
	err := number.refusal()
	switch {
	case number.Name == "":
		return fmt.Errorf("entry %d: %w", place+1, err)
	case seen[number.Name]:
		return fmt.Errorf("%s is written twice", number.Name)
	}
	seen[number.Name] = true
	if !valued && number.Says != "" {
		unvalued := fmt.Errorf("%s: a number needs a value and says "+
			"what it is", number.Name)
		return errors.Join(unvalued, err)
	}
	return err
}

// readAsNumber is whether a name parses as a number in any base, as 7,
// 010, 0x10, 1e3 and .5 do. Words such as inf and nan have no digit and
// are names.
func readAsNumber(name string) bool {
	_, intErr := strconv.ParseInt(name, fromPrefix, bits)
	_, floatErr := strconv.ParseFloat(name, bits)
	return strings.ContainsAny(name, "0123456789") &&
		(intErr == nil || floatErr == nil)
}

// refusal is why the number as filled is not one, or nothing. It does
// not judge the value, which the file may not have written.
func (number Number) refusal() error {
	switch {
	case number.Name == "":
		return errors.New("a number has no name")
	case readAsNumber(number.Name):
		return fmt.Errorf("%s: this name reads as a number, in some "+
			"base, and Get(%q) would be hard to read; call the "+
			"number something else", number.Name, number.Name)
	case number.Says == "":
		return fmt.Errorf("%s: a number needs a value and says what "+
			"it is", number.Name)
	}
	switch number.Kind {
	case Measured:
		return number.measuredRefusal()
	case Picked:
		return number.pickedRefusal()
	}
	return fmt.Errorf("%s: kind is measured or picked, not %q",
		number.Name, number.Kind)
}

// measuredRefusal is why a measured number is not one: it needs its whole
// source and no picker.
func (number Number) measuredRefusal() error {
	from := number.From
	if from == nil || from.Key == "" || from.Where == "" ||
		from.Text == "" || number.By != "" {
		return fmt.Errorf("%s: a measured number names its "+
			"source's key, where in it, and the text there, and "+
			"no one picked it", number.Name)
	}
	return nil
}

// pickedRefusal is why a picked number is not one: it needs who picked it
// and no source.
func (number Number) pickedRefusal() error {
	if number.By == "" || number.From != nil {
		return fmt.Errorf("%s: a picked number says who picked it and "+
			"when, and names no source", number.Name)
	}
	return nil
}

// Get is the number of a name, as a pointer to its place in the list, or
// nil and a refusal naming the number it does not hold.
func (read Numbers) Get(name string) (*Number, error) {
	for place := range read {
		if read[place].Name == name {
			return &read[place], nil
		}
	}
	return nil, fmt.Errorf("there is no number called %s", name)
}
