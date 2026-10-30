// Package config reads a tool's settings from a file in game's format,
// into a type the tool declares.
//
// A file is lines. A line is a key, then its words. A # as the first
// thing on a line starts a comment. Lines indented under a block's
// opening line belong to that block. A lone = after the key reads as
// nothing, so "listen = ::1" is "listen ::1".
//
// The tool declares what the file may say as a struct. NewSchema reads
// the struct's tags once, and Schema.Read then fills a value of it:
//
//	type toolFile struct {
//		Name string `config:"name,1" help:"the tool's name"`
//		Lint []Lint `config:"lint,1+" help:"a lint rule and its words"`
//	}
//
//	var schema, err = config.NewSchema[toolFile]()
//
// The type of a field says the shape of its key. See NewSchema for the
// tag and Schema for the types.
//
// A key that takes a tail has its words followed by " :: " and the rest
// of the line. The tail is trimmed at both ends, so a command or a
// reason can hold anything, :: too. An empty tail is refused as a
// missing one. For any key that takes no tail, "::" is a word.
//
// Any leading whitespace, spaces, tabs or a mix of both, of any width,
// makes a line indented, and it belongs to the block above it. The kind
// of indent is not checked. An indented line under no block is refused.
//
// Every key is declared. A file that does not conform is refused whole,
// every wrong line named by its file, line and key as "file:line: key:
// why", and the value being filled is left as it was.
//
// The file is one level deep: a block holds keys and no block.
package config
