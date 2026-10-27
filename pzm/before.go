package pzm

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	// oneWord is how many words a flag takes when its value is in the
	// word, or it has none. It's defined.
	oneWord = 1
	// twoWords is how many it takes when its value is the next word.
	// It's defined.
	twoWords = 2
	// noWords is how many words a word leaves taken when it is left to
	// the parser. It's defined.
	noWords = 0
	// first is the place of the first word. It's defined.
	first = 0
)

// some is whether a list has anything in it.
func some[item any](list []item) bool { return len(list) > noWords }

// verbFirst is args read and put in an order the parser cannot misread.
// The verb comes first. Then come the tool's flags and the verb's flags.
// The rest follows as given. Its error refuses the words before the verb.
//
// pzm reads every word before the verb itself. The parser never guesses
// which word is the verb. Each word there is a flag or the verb. The flag
// is the tool's or a verb's. It may also ask for a verb.
//
// Clusters are read letter by letter. Flags that take a value take the
// rest of their word. With nothing left, they take the next word. The
// verbs declare how each of their flags is taken, and must agree on it.
//
// Every verb flag given before the verb must belong to the verb that
// runs. Help and -- are left to the parser with the words after them.
// So are the words after the verb.
//
// With no words the result is an empty list. The parser reads the
// process's own words when it is given nil. args itself is never changed.
func (tool *Tool) verbFirst(args []string) ([]string, error) {
	read := &reading{tool: tool, tools: []string{}, verbs: []string{}}
	for at := first; at < len(args); {
		taken, how, err := read.word(args, at)
		switch {
		case err != nil:
			return nil, err
		case how == theParsers:
			return read.leftOver(args[at:])
		case how == theVerbsPlace:
			return read.named(args[at], args[at+oneWord:])
		}
		at += taken
		if read.asked != "" {
			return read.named(read.asked, args[at:])
		}
	}
	return read.leftOver(nil)
}

// how a word before the verb reads.
type how int

const (
	// aFlag is one flag or several run together. It includes any value
	// they took.
	aFlag how = iota
	// theVerbsPlace is a word that is no flag. The verb is here.
	theVerbsPlace
	// theParsers is a word left to the parser. It is help or a lone --.
	// It may be a tool flag with no value left to take.
	theParsers
)

// reading is the words before the verb, as far as they have been read.
type reading struct {
	tool  *Tool
	tools []string // the tool's flags, spelled for the parser
	verbs []string // the verbs' flags, spelled for the parser
	given []givenFlag
	aside []string // lone dashes, which are words for the verb
	asked string   // the verb asked for by its flag, if one was
}

// givenFlag is a verb's flag given before the verb: as it was typed, and
// the verbs that declare it.
type givenFlag struct {
	typed  string
	owners []string
}

// mark is how much of the reading there was before a word.
type mark struct {
	tools, verbs, given int
	asked               string
}

// marked is where the reading is now.
func (read *reading) marked() mark {
	return mark{len(read.tools), len(read.verbs), len(read.given),
		read.asked}
}

// undo puts the reading back as it was marked, since a word was left to
// the parser whole.
func (read *reading) undo(was mark) {
	read.tools = read.tools[:was.tools]
	read.verbs = read.verbs[:was.verbs]
	read.given = read.given[:was.given]
	read.asked = was.asked
}

// word reads the word at place in args. Its int is how many words it
// took, counting itself.
func (read *reading) word(args []string, place int) (int, how, error) {
	word := args[place]
	next, hasNext := wordAt(args, place+oneWord)
	if word == "--" {
		return noWords, theParsers, nil
	}
	if strings.HasPrefix(word, "--") {
		return read.long(word, next, hasNext)
	}
	if word == "-" {
		read.aside = append(read.aside, word)
		return oneWord, aFlag, nil
	}
	letters, short := strings.CutPrefix(word, "-")
	if !short {
		return noWords, theVerbsPlace, nil
	}
	return read.short(letters, next, hasNext)
}

// wordAt is the word at place in words, if there is one.
func wordAt(words []string, place int) (string, bool) {
	if place < len(words) {
		return words[place], true
	}
	return "", false
}

// long reads a long flag, --name or --name=value.
func (read *reading) long(word, next string, hasNext bool) (int, how,
	error) {
	name, _, hasValue := strings.Cut(word, "=")
	flag := read.tool.meaning(name)
	switch {
	case flag.err != nil:
		return refusal(flag.err)
	case flag.whose == nobodys:
		return refusal(fmt.Errorf("unknown flag: %s", name))
	case flag.whose == parsers || flag.whose == asksVerb && hasValue:
		return noWords, theParsers, nil
	case flag.whose == asksVerb:
		return oneWord, aFlag, read.ask(flag.verb)
	case !flag.takes || hasValue:
		read.keep(flag, name, word)
		return oneWord, aFlag, nil
	case hasNext:
		read.keep(flag, name, word+"="+next)
		return twoWords, aFlag, nil
	case flag.whose == toolsOwn:
		return noWords, theParsers, nil
	}
	read.keep(flag, name, word)
	return oneWord, aFlag, nil
}

// short reads short flags given without their dash. Switches may run
// together, as in pflag. The first flag that takes a value takes the rest
// of the word, after an = if there is one. With nothing left of the word,
// it takes the next word.
func (read *reading) short(letters, next string, hasNext bool) (int, how,
	error) {
	was := read.marked()
	for at, letter := range letters {
		name := "-" + string(letter)
		rest := letters[at+utf8.RuneLen(letter):]
		flag := read.tool.meaning(name)
		switch {
		case flag.err != nil:
			return refusal(flag.err)
		case flag.whose == nobodys:
			return refusal(unknown(letter, letters[at:]))
		case flag.whose == parsers || flag.whose == asksVerb &&
			strings.HasPrefix(rest, "="):
			read.undo(was)
			return noWords, theParsers, nil
		case flag.whose == asksVerb:
			if err := read.ask(flag.verb); err != nil {
				return refusal(err)
			}
		case !flag.takes && strings.HasPrefix(rest, "="):
			read.keep(flag, name, flag.long+rest)
			return oneWord, aFlag, nil
		case !flag.takes:
			read.keep(flag, name, flag.long)
		case rest != "":
			read.keep(flag, name, flag.long+"="+
				strings.TrimPrefix(rest, "="))
			return oneWord, aFlag, nil
		case hasNext:
			read.keep(flag, name, flag.long+"="+next)
			return twoWords, aFlag, nil
		case flag.whose == toolsOwn:
			read.undo(was)
			return noWords, theParsers, nil
		default:
			read.keep(flag, name, flag.long)
		}
	}
	return oneWord, aFlag, nil
}

// refusal is the result of a word that cannot be read. The word counts as
// taken. The reading never stalls on it, whatever its error is.
func refusal(err error) (int, how, error) { return oneWord, aFlag, err }

// unknown is the refusal of a short flag nothing declares. It says the
// rest of the cluster from that letter, as the parser does.
func unknown(letter rune, from string) error {
	return fmt.Errorf("unknown shorthand flag: %q in -%s", letter, from)
}

// ask notes a verb asked for by its flag. Each call asks for one verb.
func (read *reading) ask(name string) error {
	if read.asked != "" {
		return errors.New("one verb at a time")
	}
	read.asked = name
	return nil
}

// keep notes a flag read, in the spelling the parser is given. typed is
// the flag as the call gave it, for a refusal to name.
func (read *reading) keep(flag meaning, typed, spelled string) {
	if flag.whose == toolsOwn {
		read.tools = append(read.tools, spelled)
		return
	}
	read.verbs = append(read.verbs, spelled)
	read.given = append(read.given, givenFlag{typed, flag.owners})
}

// named is the arguments when the verb is name and rest follows it.
// Name may be no verb of the tool's. Then it goes to the parser as it is.
// The parser refuses it or answers it as help.
func (read *reading) named(name string, rest []string) ([]string, error) {
	declared := read.tool.isVerb(name)
	switch {
	case !declared && some(read.given):
		return nil, fmt.Errorf("%s is a flag of a verb, and %s is no "+
			"verb: a verb's flags come after it",
			read.given[first].typed, name)
	case !declared:
		return joined([]string{name}, read.tools, read.aside, rest), nil
	}
	for _, each := range read.given {
		if !slices.Contains(each.owners, name) {
			return nil, fmt.Errorf("%s is %s, not %s's", each.typed,
				ownedBy(each.owners), name)
		}
	}
	return joined([]string{name}, read.tools, read.verbs, read.aside,
		rest), nil
}

// joined is the lists one after another. It is never nil. The parser
// reads the process's own words when it is given nil.
func joined(lists ...[]string) []string {
	return append([]string{}, slices.Concat(lists...)...)
}

// ownedBy is the verbs that declare a flag, as a flag's refusal says it.
func ownedBy(owners []string) string {
	return strings.Join(owners, "'s and ") + "'s"
}

// leftOver is the arguments when no verb is named before rest. The parser
// is left to read rest. Any verb flag with no verb after it is refused.
// The refusal says what does come after it, when something does.
func (read *reading) leftOver(rest []string) ([]string, error) {
	switch {
	case some(read.given) && some(rest):
		return nil, fmt.Errorf("%s is a flag of a verb, and %s comes "+
			"before any verb: a verb's flags come after it",
			read.given[first].typed, rest[first])
	case some(read.given):
		return nil, fmt.Errorf("%s is a flag of a verb, and no verb "+
			"comes after it", read.given[first].typed)
	}
	return joined(read.tools, read.aside, rest), nil
}
