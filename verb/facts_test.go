package verb_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/janearc/puzzlemouth/verb"
)

// deletes are the facts of a verb that deletes in its state directory.
var deletes = verb.NewFacts(verb.WritesState, verb.Deletes,
	verb.ThisMachine, verb.NoOne, verb.Light)

// refused is why a verb with these facts cannot be made, or "".
func refused(facts verb.Facts) string {
	_, err := verb.New("foo", 'f', "h", "p", "ExampleFoo", facts)
	if err == nil {
		return ""
	}
	return err.Error()
}

// Facts with none stated are refused once for each part. Each refusal
// names the part and what to give, and all of them come at once.
func TestFactsLeftOutAreRefused(t *testing.T) {
	var unstated verb.Facts
	said := refused(unstated)
	for _, says := range []string{
		"facts: where it writes is not said: give WritesNothing, " +
			"WritesState, WritesConfig, WritesTree or WritesTo on " +
			"one line",
		"facts: what it does to what is there is not said: give " +
			"LeavesAlone, Changes or Deletes",
		"facts: whether it reaches outside the machine is not said: " +
			"give ThisMachine or Outside",
		"facts: whether it needs a person is not said: give NoOne, " +
			"Prompt, Passphrase or Confirmation",
		"facts: how hard it works the machine is not said: give " +
			"Light, FullCPU or FullGPU",
	} {
		if !strings.Contains(said, says) {
			t.Errorf("no facts: %q, want %q", said, says)
		}
	}
}

// Each part left out alone is refused alone, as is any value outside its
// kind.
func TestEachFactIsStated(t *testing.T) {
	for part, facts := range map[string]verb.Facts{
		"where it writes": verb.NewFacts("", verb.Deletes,
			verb.ThisMachine, verb.NoOne, verb.Light),
		"what it does to what is there": verb.NewFacts(verb.WritesState,
			9, verb.ThisMachine, verb.NoOne, verb.Light),
		"whether it reaches outside the machine": verb.NewFacts(
			verb.WritesState, verb.Deletes, 0, verb.NoOne, verb.Light),
		"whether it needs a person": verb.NewFacts(verb.WritesState,
			verb.Deletes, verb.ThisMachine, 9, verb.Light),
		"how hard it works the machine": verb.NewFacts(verb.WritesState,
			verb.Deletes, verb.ThisMachine, verb.NoOne, 0),
	} {
		said := refused(facts)
		if strings.Count(said, "facts:") != 1 ||
			!strings.Contains(said, part+" is not said") {
			t.Errorf("%s left out: %q", part, said)
		}
	}
}

// Facts that cannot all be true are refused. Writing nothing means
// changing and deleting nothing. Each named place is one line and holds
// no semicolon, which --agent-help puts between the facts.
func TestFactsThatCannotBeAreRefused(t *testing.T) {
	for _, each := range []struct {
		facts verb.Facts
		says  string
	}{
		{verb.NewFacts(verb.WritesNothing, verb.Deletes, verb.ThisMachine,
			verb.NoOne, verb.Light), "it writes nothing, so it cannot"},
		{verb.NewFacts(verb.WritesNothing, verb.Changes, verb.ThisMachine,
			verb.NoOne, verb.Light), "it writes nothing, so it cannot"},
		{verb.NewFacts(verb.WritesTo("a place\nand another"),
			verb.Deletes, verb.ThisMachine, verb.NoOne, verb.Light),
			"where it writes is not said"},
		{verb.NewFacts(verb.WritesTo("its state directory; the working "+
			"tree"), verb.Changes, verb.ThisMachine, verb.NoOne,
			verb.Light), "where it writes holds a semicolon, which " +
			"--agent-help puts between the facts: join two places " +
			"with and"},
		{verb.NewFacts(verb.WritesTo("nothing; leaves what is there"),
			verb.Deletes, verb.Outside, verb.Passphrase, verb.FullGPU),
			"holds a semicolon"},
	} {
		if said := refused(each.facts); !strings.Contains(said,
			each.says) {
			t.Errorf("%v: %q, want %q", each.facts, said, each.says)
		}
	}
}

// Writing anywhere changes things, and writing nothing does not.
func TestWritesAnythingIsWritingAnywhere(t *testing.T) {
	for _, place := range []verb.Writes{verb.WritesState,
		verb.WritesConfig, verb.WritesTree, verb.WritesTo("a channel")} {
		facts := verb.NewFacts(place, verb.Changes, verb.ThisMachine,
			verb.NoOne, verb.Light)
		if !facts.WritesAnything() || refused(facts) != "" {
			t.Errorf("writing %s: %q", place, refused(facts))
		}
	}
	if readsOnly.WritesAnything() {
		t.Error("writing nothing changes things")
	}
}

// Every stated fact has its words, so no line of --agent-help has a part
// missing.
func TestEveryFactHasWords(t *testing.T) {
	every := []verb.Facts{verb.NewFacts(verb.WritesNothing,
		verb.LeavesAlone, verb.ThisMachine, verb.NoOne, verb.Light)}
	for _, existing := range []verb.Existing{verb.LeavesAlone,
		verb.Changes, verb.Deletes} {
		every = append(every, verb.NewFacts(verb.WritesTree, existing,
			verb.ThisMachine, verb.NoOne, verb.Light))
	}
	for _, reaches := range []verb.Reach{verb.ThisMachine, verb.Outside} {
		every = append(every, verb.NewFacts(verb.WritesTo("x"),
			verb.Changes, reaches, verb.NoOne, verb.Light))
	}
	for _, needs := range []verb.Person{verb.NoOne, verb.Prompt,
		verb.Passphrase, verb.Confirmation} {
		every = append(every, verb.NewFacts(verb.WritesConfig,
			verb.Changes, verb.ThisMachine, needs, verb.Light))
	}
	for _, load := range []verb.Load{verb.Light, verb.FullCPU,
		verb.FullGPU} {
		every = append(every, verb.NewFacts(verb.WritesState,
			verb.Deletes, verb.ThisMachine, verb.NoOne, load))
	}
	for _, facts := range every {
		said := facts.String()
		if refused(facts) != "" || strings.Count(said, "; ") != 4 ||
			strings.Contains(said, "; ;") ||
			strings.HasSuffix(said, "; ") {
			t.Errorf("%q: %q", said, refused(facts))
		}
	}
}

// writing is a command's test file with examples that run its verb wet
// and dry, and ones that only seem to run it dry.
var writing = fstest.MapFS{"tidy_test.go": {Data: []byte(`package tidy_test

func ExampleWet() {
	run("tidy")
	// Output:
}

func ExampleDry() {
	run("tidy", "--dry-run")
	// Output:
}

func ExampleListed() {
	tool.Run([]string{"tidy", "--dry-run"}, os.Stdout, os.Stderr)
	// Output:
}

func ExampleInAHelper() {
	where.keel("tidy", "--dry-run")
	// Output:
}

func ExampleInAComment() {
	// tidy "--dry-run" would show it
	run("tidy")
	// Output:
}

func ExampleSetAside() {
	dry := "--dry-run"
	_ = dry
	run("tidy")
	// Output:
}

func ExampleInAFunction() {
	run("tidy", func() { flag := "--dry-run"; _ = flag })
	// Output:
}
`)}}

// The example of a verb that writes must pass "--dry-run" to some call,
// alone or in a list. Naming it in a comment is not that. Nor is the
// string set aside or inside a function handed to the call. The check does
// not see which verb the call runs. One that writes nothing need not.
func TestAWritingVerbsExampleMustPassDryRun(t *testing.T) {
	check := func(example verb.Example, facts verb.Facts) error {
		made, err := verb.New("tidy", 't', "h", "p", example, facts)
		if err != nil {
			t.Fatal(err)
		}
		return verb.Check(writing, made)
	}
	for _, example := range []verb.Example{"ExampleWet",
		"ExampleInAComment", "ExampleSetAside", "ExampleInAFunction"} {
		if err := check(example, deletes); err == nil ||
			!strings.Contains(err.Error(), "verb tidy: it writes its "+
				"state directory, so "+string(example)+" must pass "+
				"\"--dry-run\" to some call, and none does") {
			t.Errorf("%s: %v", example, err)
		}
	}
	for _, example := range []verb.Example{"ExampleDry",
		"ExampleListed", "ExampleInAHelper"} {
		if err := check(example, deletes); err != nil {
			t.Errorf("%s: %v", example, err)
		}
	}
	if err := check("ExampleWet", readsOnly); err != nil {
		t.Errorf("a verb that only reads: %v", err)
	}
}
