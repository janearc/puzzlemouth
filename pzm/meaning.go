package pzm

import (
	"fmt"
	"strings"
)

// whose says whose a flag is, to read it before the verb.
type whose int

const (
	// nobodys is a flag nothing declares.
	nobodys whose = iota
	// toolsOwn is a flag of the tool's, which every verb answers to.
	toolsOwn
	// asksVerb is the flag that asks for a verb: --look, or -l.
	asksVerb
	// verbsOwn is a flag that one verb or more declare.
	verbsOwn
	// parsers is help, which the parser answers.
	parsers
)

// meaning is what reading the words before the verb needs of a flag.
// That is whose it is, its long spelling and whether it takes a value.
// For a verb's flag the owners are the verbs that declare it. Any flag
// that asks for a verb has that verb.
type meaning struct {
	whose  whose
	long   string
	takes  bool
	verb   string
	owners []string
	err    error
}

// meaning is what the flag name is, as --name or -n. It looks first at
// the tool's own flags. Then it looks at the flags that ask for a verb.
// The verbs' own flags come last.
func (tool *Tool) meaning(name string) meaning {
	if name == "-h" || name == "--help" || name == "--agent-help" {
		return meaning{whose: parsers}
	}
	if flag, ok := tool.toolFlag(name); ok {
		return meaning{whose: toolsOwn, long: flag.declared.Long(),
			takes: flag.kind != switchKind}
	}
	for _, each := range tool.allVerbs() {
		if each.verb.Long() == name || each.verb.Short() == name {
			return meaning{whose: asksVerb, verb: each.verb.Name()}
		}
	}
	return tool.verbsMeaning(name)
}

// verbsMeaning is the meaning of a flag the verbs declare, or nobody's
// when none does. The verbs that declare it must agree on its long name
// and on whether it takes a value. If not, it is refused before the verb.
func (tool *Tool) verbsMeaning(name string) meaning {
	found := meaning{whose: nobodys}
	for _, each := range tool.allVerbs() {
		for _, flag := range each.allFlags() {
			if flag.declared.Long() != name &&
				flag.declared.Short() != name {
				continue
			}
			takes := flag.kind != switchKind
			unlike := found.differs(flag, takes)
			if found.whose == verbsOwn && unlike {
				found.err = disagreement(name)
			}
			found.whose, found.takes = verbsOwn, takes
			found.long = flag.declared.Long()
			found.owners = append(found.owners, each.verb.Name())
		}
	}
	return found
}

// differs is whether another verb's flag is not the one found so far.
// It takes a value where this does not. Or it goes by another long name.
// One short letter for two long names would be spelled as one of them
// for both verbs.
func (found meaning) differs(other Flag, takes bool) bool {
	return found.takes != takes || found.long != other.declared.Long()
}

// disagreement is the refusal of a flag the verbs declare differently.
func disagreement(name string) error {
	return fmt.Errorf("%s is not the same flag to every verb: put it "+
		"after the verb", name)
}

// toolFlag is the tool's own flag that word gives, if it gives one.
func (tool *Tool) toolFlag(word string) (Flag, bool) {
	name, _, _ := strings.Cut(word, "=")
	for _, each := range tool.flags {
		long, short := each.declared.Long(), each.declared.Short()
		if name == long || short != "" && name == short {
			return each, true
		}
	}
	return Flag{}, false
}

// isVerb is whether name is one of the verbs the tool answers to, its
// own or pzm's.
func (tool *Tool) isVerb(name string) bool {
	for _, each := range tool.allVerbs() {
		if each.verb.Name() == name {
			return true
		}
	}
	return false
}
