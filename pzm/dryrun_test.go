package pzm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// deletes are the facts of a verb that deletes in its state directory.
var deletes = verb.NewFacts(verb.WritesState, verb.Deletes,
	verb.ThisMachine, verb.NoOne, verb.Light)

// tidied is a tool with two verbs: tidy writes and look only reads.
func tidied() *pzm.Tool {
	tool := pzm.New("cache", "keeps what was made", "", "", "", "")
	tool.Verb("tidy", 't', "takes away what is old", "# tidy",
		"Example_tidy", deletes, tidy)
	tool.Verb("look", 'l', "says hello", "# look", "Example_hello",
		readsOnly, hello, pzm.Switch("loud", verb.NoShort, "says it louder"))
	return tool
}

// Every verb whose facts say it writes is given --dry-run, however the
// verb is asked for and wherever among its words the flag is.
func TestAVerbThatWritesIsGivenDryRun(t *testing.T) {
	for _, args := range [][]string{{"tidy", "--dry-run"},
		{"--tidy", "--dry-run"}, {"-t", "--dry-run"},
		{"tidy", "old", "--dry-run"}, {"tidy", "--dry-run=--dry-run"},
		{"--dry-run", "tidy"}, {"--dry-run", "--tidy"}} {
		code, out, errs := run(tidied(), args...)
		if code != exitOK || out != "would take away 3 old files\n" {
			t.Errorf("%q: code %d, out %q, errs %q", args, code, out,
				errs)
		}
	}
	code, out, _ := run(tidied(), "tidy")
	if code != exitOK || out != "took away 3 old files\n" {
		t.Errorf("tidy alone: code %d, out %q", code, out)
	}
}

// Where a verb writes nothing there is no --dry-run. After the verb the
// parser refuses it in its own words. Before it, the refusal says whose
// the flag is. Given twice, it is refused as any flag is.
func TestDryRunIsOnlyForAVerbThatWrites(t *testing.T) {
	for args, says := range map[string]string{
		"look --dry-run":           "cache: unknown flag: --dry-run\n",
		"--dry-run look":           "cache: --dry-run is tidy's, not look's\n",
		"--dry-run -l":             "cache: --dry-run is tidy's, not look's\n",
		"tidy --dry-run --dry-run": "it is given twice",
	} {
		code, out, errs := run(tidied(), strings.Fields(args)...)
		if code != exitRefused || out != "" || !strings.Contains(errs, says) {
			t.Errorf("%s: code %d, out %q, errs %q", args, code, out,
				errs)
		}
	}
}

// --help shows --dry-run for a verb that writes and not for one that
// only reads; --agent-help says each verb's facts and the flag.
func TestTheHelpSaysDryRunAndTheFacts(t *testing.T) {
	const asks = "asks the verb to say what it would do, and do none of it"
	const dry = "--dry-run    " + asks
	_, tidyHelp, _ := run(tidied(), "tidy", "--help")
	_, lookHelp, _ := run(tidied(), "look", "--help")
	if !strings.Contains(tidyHelp, dry) ||
		strings.Contains(lookHelp, "--dry-run") {
		t.Errorf("tidy:\n%s\nlook:\n%s", tidyHelp, lookHelp)
	}
	_, agent, _ := run(tidied(), "--agent-help")
	for _, line := range []string{
		"verb tidy -t: takes away what is old\nfacts tidy: writes its " +
			"state directory; may delete what is there; stays on " +
			"this machine; needs no one; light on the machine\n" +
			"flag tidy --dry-run: " + asks + "\n",
		"facts look: " + readOnlyWords + "\nflag look --loud: says " +
			"it louder\nverb age",
	} {
		if !strings.Contains(agent, line) {
			t.Errorf("--agent-help has no %q:\n%s", line, agent)
		}
	}
}

// The tool's examples are held: tidy's shows its dry run.
func TestTheDryRunExampleIsHeld(t *testing.T) {
	if err := tidied().Examples(os.DirFS(".")); err != nil {
		t.Error(err)
	}
}
