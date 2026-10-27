package pzm_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// toolHelp is what --help says of the test tool: its line, one line per
// verb, and how to call it.
const toolHelp = `demo: a tool for testing pzm

  foo-bar    -f   foos the bar
  baz        -b   fails, always
  quiet           has no short flag

demo VERB [ARGS], or demo --VERB [ARGS]
--help for this, --agent-help for the same, for an agent
--age and --who for which build this is, and who made it
`

// helped is the test tool with a verb that has no short flag, to show
// the column it leaves empty.
func helped() *pzm.Tool {
	tool, _ := built()
	tool.Verb("quiet", verb.NoShort, "has no short flag", "# quiet", "ExampleQuiet", readsOnly,
		func(pzm.Call) error { return nil })
	return tool
}

// --help, -h, help, and nothing at all, each say the tool's help.
func TestTheToolsHelp(t *testing.T) {
	for _, asked := range [][]string{{"--help"}, {"-h"}, {"help"}, {}} {
		code, out, errs := run(helped(), asked...)
		if code != exitOK || out != toolHelp {
			t.Errorf("%v: code %d, errs %q, help:\n%s", asked, code, errs, out)
		}
	}
}

// A verb's own help says what it does and every way to ask for it.
func TestAVerbsHelp(t *testing.T) {
	cases := map[string][]string{
		"demo foo-bar [ARGS]: foos the bar\n" +
			"the same as demo --foo-bar or demo -f\n": {"foo-bar", "--help"},
		"demo quiet [ARGS]: has no short flag\n" +
			"the same as demo --quiet\n": {"help", "quiet"},
	}
	for want, asked := range cases {
		code, out, errs := run(helped(), asked...)
		if code != exitOK || out != want {
			t.Errorf("%v: code %d, errs %q, help:\n%s", asked, code, errs, out)
		}
	}
}

// agentHelp is what --agent-help says of the test tool: one labelled line
// per thing and how it exits. Each verb's facts have their own line.
const agentHelp = `tool demo: a tool for testing pzm
verb foo-bar -f: foos the bar
facts foo-bar: ` + readOnlyWords + `
verb baz -b: fails, always
facts baz: ` + readOnlyWords + `
verb quiet: has no short flag
facts quiet: ` + readOnlyWords + `
verb age: which commit this is, and how long ago it was made
facts age: ` + readOnlyWords + `
verb who: who built this, and the sessions behind it
facts who: ` + readOnlyWords + `
call demo VERB [ARGS] or demo --VERB [ARGS]; exit 0 ran, 1 the verb failed, 2 nothing ran
`

// readOnlyWords are the facts of a verb that only reads, in words.
const readOnlyWords = "writes nothing; leaves what is there; stays on " +
	"this machine; needs no one; light on the machine"

// --agent-help says the tool to an agent, and is one verb at a time like
// any other.
func TestTheToolsAgentHelp(t *testing.T) {
	code, out, errs := run(helped(), "--agent-help")
	if code != exitOK || out != agentHelp {
		t.Errorf("code %d, errs %q, agent help:\n%s", code, errs, out)
	}
	if code, _, _ := run(helped(), "--agent-help", "--foo-bar"); code != exitRefused {
		t.Errorf("--agent-help with --foo-bar: code %d", code)
	}
}

// The names pzm answers to itself cannot be a tool's verbs.
func TestPzmsOwnNamesAreTaken(t *testing.T) {
	for _, name := range []string{"help", "agent-help", "age", "who"} {
		tool, _ := built()
		tool.Verb(verb.Name(name), verb.NoShort, "h", "p", "ExampleTaken", readsOnly,
			func(pzm.Call) error { return nil })
		if err := tool.Check(); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: Check said %v", name, err)
		}
	}
}
