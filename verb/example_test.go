package verb_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/janearc/puzzlemouth/verb"
)

// age is the verb the example declares, so the test below can check the
// example against this package's own tests.
func age() (*verb.Verb, error) {
	return verb.New("age", verb.NoShort,
		"which commit this binary is, and how long ago it was made",
		"# age\n\nwhich commit this binary was built from.", "ExampleNew",
		verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
			verb.ThisMachine, verb.NoOne, verb.Light))
}

// A command declares a verb once, and every way of asking about it reads
// from that one declaration.
func ExampleNew() {
	declared, err := age()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(declared.Long(), "|", declared.Help())
	// Output: --age | which commit this binary is, and how long ago it was made
}

// Facts say what a verb does to the machine. --agent-help says them in
// these words, and a verb that writes is given --dry-run.
func ExampleFacts() {
	tidy := verb.NewFacts(verb.WritesState, verb.Deletes,
		verb.ThisMachine, verb.NoOne, verb.Light)
	fmt.Println(tidy)
	fmt.Println(tidy.WritesAnything())
	// Output:
	// writes its state directory; may delete what is there; stays on this machine; needs no one; light on the machine
	// true
}

// verb's own example passes the check it asks of every command, read
// from this directory the way a command reads its embedded tests.
func TestTheExampleHere(t *testing.T) {
	declared, err := age()
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Check(os.DirFS("."), declared); err != nil {
		t.Error(err)
	}
}
