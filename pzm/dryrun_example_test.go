package pzm_test

import (
	"fmt"
	"os"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// tidy is the verb doc/facts.md declares. It takes away what is old in
// its state directory, and on a dry run says what it would take away.
func tidy(call pzm.Call) error {
	if call.DryRun() {
		_, err := fmt.Fprintln(call.Out(), "would take away 3 old files")
		return err
	}
	_, err := fmt.Fprintln(call.Out(), "took away 3 old files")
	return err
}

// doc/facts.md's sample: a verb whose facts say it deletes in its state
// directory. pzm gives it --dry-run, and its example runs it both ways.
func Example_tidy() {
	tool := pzm.New("cache", "keeps what was made", "", "", "", "")
	page := verb.Page("# tidy\n\ntakes away what is old.")
	facts := verb.NewFacts(verb.WritesState, verb.Deletes,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("tidy", verb.NoShort, "takes away what is old", page, "Example_tidy",
		facts, tidy)
	tool.Run([]string{"tidy", "--dry-run"}, os.Stdout, os.Stderr)
	tool.Run([]string{"tidy"}, os.Stdout, os.Stderr)
	// Output:
	// would take away 3 old files
	// took away 3 old files
}
