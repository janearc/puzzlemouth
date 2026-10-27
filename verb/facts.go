package verb

import (
	"fmt"
	"strings"
)

// Facts are what a verb does to the machine it runs on. They are typed
// so that every tool tells a person and an agent the same things the
// same way. NewFacts makes them.
type Facts struct {
	writes   Writes
	existing Existing
	reaches  Reach
	needs    Person
	load     Load
}

// NewFacts are a verb's facts. The parts come in the order their types
// are declared below, from Writes to Load. Each type is its own, so two
// parts out of order do not compile. New refuses a part not stated and
// two parts that cannot both be true.
func NewFacts(writes Writes, existing Existing, reaches Reach,
	needs Person, load Load) Facts {
	return Facts{writes, existing, reaches, needs, load}
}

// Writes is where a verb writes: one of the places below, or one that
// WritesTo names.
type Writes string

// The places a verb writes that every tool has.
const (
	WritesNothing Writes = "nothing"
	WritesState   Writes = "its state directory"
	WritesConfig  Writes = "its config directory"
	WritesTree    Writes = "the working tree"
)

// WritesTo names any other place a verb writes, in a few words on one
// line: "the agents' homes" or "a slack channel".
func WritesTo(place string) Writes { return Writes(place) }

// Existing is what a verb does to what is already there.
type Existing int

// What a verb may do to what is already there.
const (
	// The values start above zero so the zero value is not a statement.
	// It's defined.
	LeavesAlone Existing = iota + 1 // adds, if anything
	Changes                         // may change it
	Deletes                         // may delete it
)

// Reach is whether a verb reaches outside the machine.
type Reach int

// Where a verb reaches.
const (
	// The values start above zero so the zero value is not a statement.
	// It's defined.
	ThisMachine Reach = iota + 1 // stays on it
	Outside                      // reaches outside it
)

// Person is whether a verb needs a person while it runs.
type Person int

// What a verb needs of a person.
const (
	// The values start above zero so the zero value is not a statement.
	// It's defined.
	NoOne        Person = iota + 1 // runs alone
	Prompt                         // asks a question
	Passphrase                     // asks for a passphrase
	Confirmation                   // asks to go ahead
)

// Load is how hard a verb can work the machine.
type Load int

// How hard a verb works the machine.
const (
	// The values start above zero so the zero value is not a statement.
	// It's defined.
	Light   Load = iota + 1 // never holds a core
	FullCPU                 // can hold a cpu at full load
	FullGPU                 // can hold the gpu at full load
)

// said are the words for each stated fact, in the order they are given.
var said = struct {
	existing map[Existing]string
	reaches  map[Reach]string
	needs    map[Person]string
	load     map[Load]string
}{
	existing: map[Existing]string{LeavesAlone: "leaves what is there",
		Changes: "may change what is there",
		Deletes: "may delete what is there"},
	reaches: map[Reach]string{ThisMachine: "stays on this machine",
		Outside: "reaches outside this machine"},
	needs: map[Person]string{NoOne: "needs no one",
		Prompt:       "asks a person a question",
		Passphrase:   "asks a person for a passphrase",
		Confirmation: "asks a person to go ahead"},
	load: map[Load]string{Light: "light on the machine",
		FullCPU: "can hold a cpu at full load",
		FullGPU: "can hold the gpu at full load"},
}

// WritesAnything is whether the verb writes anywhere. pzm gives every verb
// that does --dry-run.
func (facts Facts) WritesAnything() bool {
	return facts.writes != WritesNothing
}

// String is the facts in words, each part after a semicolon.
func (facts Facts) String() string {
	return strings.Join([]string{"writes " + string(facts.writes),
		said.existing[facts.existing], said.reaches[facts.reaches],
		said.needs[facts.needs], said.load[facts.load]}, "; ")
}

// refusals are why facts cannot be what they say: one not stated, or
// two that cannot both be true.
func (facts Facts) refusals() []string {
	missing := &parts{}
	unsaid := missing.say
	_, existing := said.existing[facts.existing]
	_, reaches := said.reaches[facts.reaches]
	_, needs := said.needs[facts.needs]
	_, load := said.load[facts.load]
	place := string(facts.writes)
	unsaid(oneLine(place), "where it writes",
		"WritesNothing, WritesState, WritesConfig, WritesTree or "+
			"WritesTo on one line")
	unsaid(existing, "what it does to what is there",
		"LeavesAlone, Changes or Deletes")
	unsaid(reaches, "whether it reaches outside the machine",
		"ThisMachine or Outside")
	unsaid(needs, "whether it needs a person",
		"NoOne, Prompt, Passphrase or Confirmation")
	unsaid(load, "how hard it works the machine",
		"Light, FullCPU or FullGPU")
	if !facts.WritesAnything() && existing &&
		facts.existing != LeavesAlone {
		missing.refused = append(missing.refused, "it writes nothing, "+
			"so it cannot change or delete what is there")
	}
	if strings.Contains(string(facts.writes), ";") {
		missing.refused = append(missing.refused, "where it writes "+
			"holds a semicolon, which --agent-help puts between "+
			"the facts: join two places with and")
	}
	return missing.refused
}

// parts are what a fact set leaves unsaid or contradicts.
type parts struct{ refused []string }

// say keeps the part as refused when it is not stated.
func (missing *parts) say(stated bool, part, give string) {
	if !stated {
		missing.refused = append(missing.refused, fmt.Sprintf(
			"%s is not said: give %s", part, give))
	}
}
