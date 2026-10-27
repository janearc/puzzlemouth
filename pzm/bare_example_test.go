package pzm_test

import (
	"fmt"
	"io"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// look says what it sees, and changes nothing: the kind of verb a tool
// can run bare.
func look(call pzm.Call) error {
	_, err := fmt.Fprintln(call.Out(), "nothing out of place")
	return err
}

// doc/bare.md's sample: a tool that runs its one verb when it is called
// with nothing after it. --help still prints its help, and says so.
func Example_look() {
	tool := pzm.New("peek", "looks without touching", "", "", "", "")
	// as at a terminal, which a test's output never is
	tool.AtTerminal(func(io.Reader, io.Writer) bool { return true })
	page := verb.Page("# look\n\nsays what is there.")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("look", verb.NoShort, "says what is there", page, "Example_look", facts,
		look)
	tool.Bare("look")
	tool.Run(nil, os.Stdout, os.Stderr)
	tool.Run([]string{"--help"}, os.Stdout, os.Stderr)
	// Output:
	// nothing out of place
	// peek: looks without touching
	//
	//   look         says what is there
	//
	// peek VERB [ARGS], or peek --VERB [ARGS]
	// peek with nothing after it, at a terminal, is peek look
	// --help for this, --agent-help for the same, for an agent
	// --age and --who for which build this is, and who made it
}
