// Package yamlfile is the one way a pzm program reads a YAML file. The
// caller's type says the shape: Read fills a value of it from the file,
// by the yaml tags on its fields.
//
// It is for data files such as numbers.yaml and sources.yaml. Tool
// settings are not YAML; they are read by package config.
//
// A file is taken whole or not at all. It is refused for a key the type
// has no field for, a key written twice, a value of the wrong type for
// its field, and text that is not YAML.
//
// Every refusal names the file, and the line wherever the YAML library
// gives one. It gives one for all but some faults of syntax.
package yamlfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Read fills into from raw, the text of the file called name. The name is
// only for the errors; nothing is opened.
//
// Fields are matched by their yaml tags. A field with no tag is matched by
// its name in lower case. A field tagged "-" is not part of the file.
//
// Read refuses, all at once where the YAML library allows it:
//
//   - a key that no field of the type has;
//   - a value that cannot be the type of its field, such as hello for an
//     int;
//   - text that is not YAML at all, which stops at the first fault the
//     library finds;
//   - a key written twice in one mapping, which also stops there: the
//     library reports the first twice-written key and nothing after it.
//
// The error holds the file name once, then one refusal to a line, each as
// "line N: why". Read returns nothing but the error, and when it refuses,
// into is untouched, as config's Schema.Read leaves it.
//
// Only what the file says is written to into. A key the file leaves out
// keeps the value into held, so a caller sets its defaults first. An empty
// file, or one of only comments, is not a refusal and leaves into as it
// was.
//
// Scalars are taken as written, whatever they look like: yes stays the
// string yes and 010 the string 010 when the field is a string. Read does
// not turn them into true or 8. It is the caller's to refuse a name it
// finds confusing.
//
// Read reads the first YAML document of raw and refuses a file with a
// second one. It refuses a nil into.
//
// A pointer or a map that the caller set in into before is filled in
// place, so a refused file may leave it part filled. Every other field is
// as the caller had it.
func Read[T any](name string, raw []byte, into *T) error {
	if into == nil {
		return fmt.Errorf("%s: there is nowhere to read the file into",
			name)
	}
	work := *into
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	err := decoder.Decode(&work)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return refusal(name, reasons(err))
	}
	var second yaml.Node
	if decoder.Decode(&second) == nil {
		why := fmt.Sprintf("line %d: a second document; a file holds "+
			"one", second.Content[first].Line)
		return refusal(name, []string{why})
	}
	*into = work
	return nil
}

// first is the first place in a list. It's defined.
const first = 0

// reasons are the refusals the YAML library gives, one to a line, each
// beginning with its line where it has one.
func reasons(err error) []string {
	var typed *yaml.TypeError
	if errors.As(err, &typed) {
		return typed.Errors
	}
	return []string{strings.TrimPrefix(err.Error(), "yaml: ")}
}

// refusal is the file's name once, then each reason on a line of its own.
func refusal(name string, why []string) error {
	return fmt.Errorf("%s:\n  %s", name, strings.Join(why, "\n  "))
}
