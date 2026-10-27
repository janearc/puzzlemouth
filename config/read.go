package config

import (
	"errors"
	"fmt"
	"strings"
)

// reading is one file being read. It keeps the file's name and what it
// may set. It keeps the entries so far and every line it bounced. While a
// block is open it keeps that entry and the block's settings.
type reading struct {
	name    string
	top     []Setting
	read    []Entry
	opened  *Entry
	inner   []Setting
	bounced []error
}

// Read is a file's entries, checked against the settings declared, or
// the file bounced: nothing is read from it, and the error names every
// line that does not conform, by name, the file's, and line.
//
// Settings declared with a key twice are refused before anything is read.
func Read(name string, raw []byte, settings ...Setting) ([]Entry, error) {
	if err := settingsRefusal(settings); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	file := &reading{name: name, top: settings}
	for at, text := range strings.Split(string(raw), "\n") {
		file.line(at+one, strings.TrimRight(text, "\r"))
	}
	if len(file.bounced) > none {
		return nil, errors.Join(file.bounced...)
	}
	return file.read, nil
}

// line reads one line: a comment or a blank, a setting, or a setting
// inside the block above it.
func (file *reading) line(at int, text string) {
	trimmed := strings.TrimLeft(text, " \t")
	if strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "#") {
		return
	}
	indent := text[:len(text)-len(trimmed)]
	if strings.Contains(indent, "\t") {
		file.bounce(at, "it is indented with a tab; indent with spaces")
		return
	}
	if indent == "" {
		file.opened = nil
		each, setting, ok := file.entry(at, trimmed, file.top, "")
		if ok {
			file.read = append(file.read, each)
			if setting.block {
				file.opened = &file.read[len(file.read)-one]
				file.inner = setting.inner
			}
		}
		return
	}
	if file.opened == nil {
		file.bounce(at, "it is indented, but under no block")
		return
	}
	each, _, ok := file.entry(at, trimmed, file.inner, file.opened.key)
	if ok {
		file.opened.in = append(file.opened.in, each)
	}
}

const (
	// none is the count of nothing. It's defined.
	none = 0
	// one is a count of one, and the step from a length to the last place.
	// It's defined.
	one = 1
	// first is the first place in a list. It's defined.
	first = 0
	// keyWords is how many words come before a setting's own: its key.
	// It's defined.
	keyWords = 1
	// noKey is why a line is bounced when nothing it can read as a key
	// comes before its divider. Spaces of another kind can do that.
	noKey = `there is no setting before the "::"`
)

// entry is one line as the setting its key names, when it is one of
// those that may be set here and the line takes what it declares.
func (file *reading) entry(at int, text string, may []Setting,
	inside string) (Entry, Setting, bool) {
	head, tail, tailed := strings.Cut(text, divider)
	tail = strings.TrimSpace(tail)
	words := strings.Fields(head)
	if len(words) < keyWords {
		file.bounce(at, noKey)
		return Entry{}, Setting{}, false
	}
	key := words[first]
	words = words[one:]
	if len(words) > none && words[first] == "=" {
		words = words[one:]
	}
	setting, known := find(may, key)
	if !known {
		file.bounce(at, unknown(key, may, inside))
		return Entry{}, Setting{}, false
	}
	words, tailed = emptyTail(setting, words, tailed)
	if why := setting.takes(len(words), tailed, tail); why != "" {
		file.bounce(at, why)
		return Entry{}, Setting{}, false
	}
	return Entry{key: key, words: words, tail: tail, line: at}, setting,
		true
}

// emptyTail is a line's words without a last "::", when its setting
// takes a tail and the line ends in one: a tail with nothing in it. For
// any other setting "::" is a word, such as an address to listen on.
func emptyTail(setting Setting, words []string, tailed bool) ([]string,
	bool) {
	last := len(words) - one
	if tailed || setting.tail == NoTail || last < none ||
		words[last] != "::" {
		return words, tailed
	}
	return words[:last], true
}

// takes is why a line does not take what its setting declares, or
// nothing when it does.
func (setting Setting) takes(words int, tailed bool, tail string) string {
	switch {
	case words != int(setting.words):
		wanted := counted(int(setting.words))
		return fmt.Sprintf("%s takes %s, and this line has %d",
			setting.key, wanted, words)
	case tailed && setting.tail == NoTail:
		return fmt.Sprintf("%s takes no tail after %q", setting.key,
			strings.TrimSpace(divider))
	case setting.tail == Tail && tail == "":
		return fmt.Sprintf("%s takes a tail after %q", setting.key,
			strings.TrimSpace(divider))
	}
	return ""
}

// counted is a number of words, said.
func counted(words int) string {
	if words == one {
		return "1 word"
	}
	return fmt.Sprintf("%d words", words)
}

// find is the setting a key names among those that may be set.
func find(may []Setting, key string) (Setting, bool) {
	for _, setting := range may {
		if string(setting.key) == key {
			return setting, true
		}
	}
	return Setting{}, false
}

// unknown is why a key that is not a setting here is bounced, with the
// settings there are.
func unknown(key string, may []Setting, inside string) string {
	var names []string
	for _, setting := range may {
		names = append(names, string(setting.key))
	}
	if inside != "" {
		return fmt.Sprintf("%q is not a setting of %s; "+
			"its settings are %s", key, inside,
			strings.Join(names, ", "))
	}
	return fmt.Sprintf("%q is not a setting; the settings are %s", key,
		strings.Join(names, ", "))
}

// bounce keeps why a line does not conform, by the file's name and the
// line.
func (file *reading) bounce(at int, why string) {
	file.bounced = append(file.bounced,
		fmt.Errorf("%s:%d: %s", file.name, at, why))
}
