package pzm

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/janearc/puzzlemouth/verb"
)

// kind is how a flag takes its value. Switches take none and values take
// one word; lists take one word and split it on commas. --help, refusals
// and doc/flags.md name the kinds the same way.
type kind int

const (
	// switchKind is on when it is given, and takes no value: --loud.
	switchKind kind = iota
	// valueKind takes one value: --name ada, or --name=ada.
	valueKind
	// listKind takes one value split on its commas: --to ada,bo.
	listKind
)

// String is the kind's name.
func (each kind) String() string {
	return [...]string{"switch", "value", "list"}[each]
}

// Flag is one flag of a verb or of the whole tool. Switch, Value and List
// make one. One that cannot be made keeps why, and then the tool does not
// run.
type Flag struct {
	declared verb.Flag
	kind     kind
	err      error
}

// Switch is a flag that is on when given, and takes no value: --loud.
func Switch(name verb.Name, short verb.Short, help verb.Help) Flag {
	return newFlag(name, short, help, switchKind)
}

// Value is a flag that takes one value: --name ada, or --name=ada.
func Value(name verb.Name, short verb.Short, help verb.Help) Flag {
	return newFlag(name, short, help, valueKind)
}

// List is a flag that takes one value split on its commas: --to ada,bo.
func List(name verb.Name, short verb.Short, help verb.Help) Flag {
	return newFlag(name, short, help, listKind)
}

// newFlag is a flag of a kind, or why it cannot be.
func newFlag(name verb.Name, short verb.Short, help verb.Help,
	how kind) Flag {
	declared, err := verb.NewFlag(verb.Unlimited, name, short, help)
	return Flag{declared, how, err}
}

// flagValue is what one run of the tool gave a flag, as the parser
// underneath sets it.
type flagValue struct {
	flag  Flag
	given bool
	text  string
	items []string
}

// errTwice is why a flag given a second time is refused: which of the two
// was meant is not pzm's to guess.
var errTwice = errors.New("it is given twice")

// Set takes the flag's value as the command line gives it.
//
// Lists split the value on commas. The empty value is the empty list,
// and an empty item is refused.
//
// Switches take no value. The parser hands a bare switch its own long
// form, such as --loud. So a switch given any value is told apart and
// refused, "true" among them. Then a refusal that quotes the value quotes
// the flag.
func (value *flagValue) Set(text string) error {
	switch {
	case value.given:
		return errTwice
	case value.flag.kind == switchKind &&
		text != value.flag.declared.Long():
		return errors.New("a switch takes no value")
	}
	value.given = true
	value.text = text
	if value.flag.kind != listKind {
		return nil
	}
	value.items = []string{}
	if text != "" {
		value.items = strings.Split(text, ",")
	}
	if slices.Contains(value.items, "") {
		return fmt.Errorf("%q has an empty item", text)
	}
	return nil
}

// String is the value as given, or nothing.
func (value *flagValue) String() string { return value.text }

// Type is the flag's kind.
func (value *flagValue) Type() string { return value.flag.kind.String() }

// On is whether the switch name was given.
func (call Call) On(name string) bool {
	return call.flag(name, switchKind).given
}

// Value is the value given to the flag name, or "" when it was not
// given; Given tells "" given from not given.
func (call Call) Value(name string) string {
	return call.flag(name, valueKind).text
}

// List is the items given to the list flag name. The empty value gives
// none, and a flag not given gives nil.
func (call Call) List(name string) []string {
	return call.flag(name, listKind).items
}

// Given is whether the flag name was given, whatever its kind.
func (call Call) Given(name string) bool {
	if value, ok := call.flags[name]; ok {
		return value.given
	}
	panic(call.undeclared(name, "flag"))
}

// flag is the flag name as this run was given it. Asking for a flag the
// verb did not declare is a bug in the tool, and so is asking for it as
// another kind. The panic says which.
func (call Call) flag(name string, want kind) *flagValue {
	value, ok := call.flags[name]
	if !ok || value.flag.kind != want {
		asked := want.String()
		panic(call.undeclared(name, asked))
	}
	return value
}

// undeclared is what a verb that asked for a flag it does not have is
// told.
func (call Call) undeclared(name, what string) string {
	return fmt.Sprintf("pzm: %s asks for --%s, which it does not declare "+
		"as a %s", call.verb, name, what)
}
