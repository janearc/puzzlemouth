// Package config reads a tool's settings from a file in game's format.
// Each setting is one line: its key, then its words. The # as the first thing
// on a line starts a comment. Block lines are indented under it.
//
// Any setting declared with a tail takes the rest of the line after " :: ".
// The tail is trimmed at both ends, so a command or a reason can hold
// anything, :: too. An empty tail is bounced as a missing one. Any lone =
// after the key reads as nothing, so "listen = ::1" is "listen ::1".
//
// Any line with " :: " and no key before it is bounced.
//
// Every setting is declared. Any file that does not conform is bounced
// whole, and every wrong line is named by its file and line.
//
// The file is one level deep: a block holds settings and no block.
// Declaring a key twice, or a block inside a block, is refused. Read
// refuses it too, before it reads.
package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Key is a setting's name, lowercase words joined by hyphens.
type Key string

// Help is what a setting is for, in one line.
type Help string

// Words is how many words a setting takes after its key.
type Words int

// Takes is whether a setting takes a tail after " :: ".
type Takes bool

const (
	// NoTail is a setting that ends with its words.
	NoTail Takes = false
	// Tail is a setting whose words are followed by " :: " and the rest
	// of the line.
	Tail Takes = true
)

// divider is what comes between a setting's words and its tail.
const divider = " :: "

// Setting is one thing a file may set. It has a key and a help line and
// what it takes. Any block also has the settings its indented lines may
// set.
type Setting struct {
	key   Key
	help  Help
	words Words
	tail  Takes
	block bool
	inner []Setting
}

// keyed is what a key may be.
var keyed = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// New is a setting, or why it cannot be one.
func New(key Key, help Help, words Words, tail Takes) (Setting, error) {
	var refused []error
	if !keyed.MatchString(string(key)) {
		refused = append(refused, fmt.Errorf(
			"%q is not lowercase words joined by hyphens", key))
	}
	if strings.TrimSpace(string(help)) == "" ||
		strings.ContainsAny(string(help), "\r\n") {
		refused = append(refused,
			errors.New("its help is not one line"))
	}
	if words < none || words == none && tail == NoTail {
		refused = append(refused, errors.New(
			"it takes no words and no tail, so it can say nothing"))
	}
	if len(refused) > none {
		return Setting{}, fmt.Errorf("setting %s: %w", key,
			errors.Join(refused...))
	}
	return Setting{key: key, help: help, words: words, tail: tail}, nil
}

// Block is a setting whose line opens a block. The lines indented under
// it set the settings it holds.
//
// The file is one level deep. Each block holds no block and no key twice.
// The settings are copied, so the caller may change them after.
func Block(key Key, help Help, words Words, inner ...Setting) (Setting,
	error) {
	made, err := New(key, help, words, NoTail)
	if err == nil {
		err = innerRefusal(key, inner)
	}
	if err != nil {
		return Setting{}, err
	}
	made.block, made.inner = true, slices.Clone(inner)
	return made, nil
}

// innerRefusal is why a block cannot hold inner. Inner must not be empty.
// It must not hold a block, since a file is one level deep. It must not
// hold settings that Read would refuse.
func innerRefusal(key Key, inner []Setting) error {
	if len(inner) == none {
		return fmt.Errorf("setting %s: a block that holds nothing", key)
	}
	refused := []error{settingsRefusal(inner)}
	for _, each := range inner {
		if each.block {
			refused = append(refused, fmt.Errorf("%s is a block; "+
				"a file is one level deep", each.key))
		}
	}
	if err := errors.Join(refused...); err != nil {
		return fmt.Errorf("setting %s: %w", key, err)
	}
	return nil
}

// errNotMade is why a setting that is the zero value is refused.
var errNotMade = errors.New("a setting that config.New or " +
	"config.Block did not make")

// settingsRefusal is why settings cannot be declared together. One was
// not made by New or Block, or two share a key.
func settingsRefusal(settings []Setting) error {
	var refused []error
	seen := map[Key]bool{}
	for _, each := range settings {
		switch {
		case each.key == "":
			refused = append(refused, errNotMade)
		case seen[each.key]:
			refused = append(refused, fmt.Errorf("%s is declared "+
				"twice", each.key))
		}
		seen[each.key] = true
	}
	return errors.Join(refused...)
}

// Entry is one line a file set. It has a key and words and a tail and
// a line number. Any block entry also has the lines under it.
type Entry struct {
	key   string
	words []string
	tail  string
	line  int
	in    []Entry
}

// Key is the setting the line set.
func (each Entry) Key() string { return each.key }

// Words are the words after the key.
func (each Entry) Words() []string { return each.words }

// Tail is the rest of the line after " :: ", or nothing.
func (each Entry) Tail() string { return each.tail }

// Line is the line of the file the entry is on, counted from one.
func (each Entry) Line() int { return each.line }

// In is the lines indented under a block, in order.
func (each Entry) In() []Entry { return each.in }
