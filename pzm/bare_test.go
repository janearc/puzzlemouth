package pzm_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// peek runs look bare. Its other verb is tidy, which writes. So a test
// can tell which ran. look fails when a test asks it to.
func peek(failing bool) *pzm.Tool {
	tool := pzm.New("peek", "looks without touching", "", "", "", "")
	tool.Flags(pzm.Value("in", verb.NoShort, "where to look"))
	tool.Verb("look", 'l', "says what is there", "# look", "Example_look",
		readsOnly, func(call pzm.Call) error {
			if failing {
				return errors.New("could not see")
			}
			return look(call)
		})
	tool.Verb("tidy", verb.NoShort, "puts things away", "# tidy", "Example_tidy",
		verb.NewFacts(verb.WritesTo("the room"), verb.Changes,
			verb.ThisMachine, verb.NoOne, verb.Light),
		func(call pzm.Call) error {
			said := "tidied"
			if call.DryRun() {
				said = "would tidy"
			}
			_, err := fmt.Fprintln(call.Out(), said)
			return err
		})
	tool.Bare("look")
	tool.AtTerminal(atATerminal)
	return tool
}

// Called bare, peek runs the verb named. Its exit code is that verb's: 0
// when it runs and 1 when it fails. Words after the tool read as before.
func TestABareCallRunsTheVerbNamed(t *testing.T) {
	for _, each := range []struct {
		args    []string
		failing bool
		code    int
		out     string
		errs    string
	}{
		{nil, false, 0, "nothing out of place\n", ""},
		{[]string{}, false, 0, "nothing out of place\n", ""},
		{nil, true, 1, "", "peek look: could not see\n"},
		{[]string{"look"}, false, 0, "nothing out of place\n", ""},
		{[]string{"tidy"}, false, 0, "tidied\n", ""},
		{[]string{"tidy", "--dry-run"}, false, 0, "would tidy\n", ""},
		{[]string{"stray"}, false, 2, "", "peek: there is no verb stray\n"},
		{[]string{"--in", "attic"}, false, 2, "", ""},
	} {
		var out, errs strings.Builder
		code := peek(each.failing).Run(each.args, &out, &errs)
		if code != each.code || out.String() != each.out ||
			each.errs != "" && errs.String() != each.errs {
			t.Errorf("peek %q: code %d, out %q, errs %q; want %d, %q, %q",
				each.args, code, out.String(), errs.String(), each.code,
				each.out, each.errs)
		}
	}
}

// The tool's own flags with no verb are still refused. Given --in and
// nothing else, the tool has been asked something it cannot tell.
func TestToolFlagsAloneAreStillRefused(t *testing.T) {
	var out, errs strings.Builder
	code := peek(false).Run([]string{"--in", "attic"}, &out, &errs)
	if code != exitRefused || out.Len() != none || !strings.Contains(errs.String(),
		"no verb") {
		t.Errorf("peek --in attic: code %d, out %q, errs %q", code,
			out.String(), errs.String())
	}
}

// --help and --agent-help still print help, and each says what a bare
// call runs; a tool that names none says nothing of it.
func TestHelpSaysWhatABareCallRuns(t *testing.T) {
	for _, each := range []struct {
		args []string
		says string
	}{
		{[]string{"--help"}, "peek with nothing after it, at a terminal, " +
			"is peek look\n"},
		{[]string{"--agent-help"}, "bare peek: runs look at a terminal, " +
			"and prints --help elsewhere\n"},
	} {
		var out, errs strings.Builder
		if code := peek(false).Run(each.args, &out, &errs); code != exitOK ||
			!strings.Contains(out.String(), each.says) {
			t.Errorf("peek %q: code %d, out %q", each.args, code,
				out.String())
		}
	}
	plain := pzm.New("hi", "says hello", "", "", "", "")
	plain.Verb("hello", verb.NoShort, "says hello", "# hello", "Example_hello",
		readsOnly, hello)
	for _, args := range [][]string{nil, {"--help"}, {"--agent-help"}} {
		var out, errs strings.Builder
		if code := plain.Run(args, &out, &errs); code != exitOK ||
			strings.Contains(out.String(), "bare") ||
			strings.Contains(out.String(), "nothing after it") {
			t.Errorf("hi %q: code %d, out %q", args, code, out.String())
		}
	}
	var out, errs strings.Builder
	plain.Run(nil, &out, &errs)
	if !strings.HasPrefix(out.String(), "hi: says hello\n") {
		t.Errorf("hi with nothing after it printed %q, not its help",
			out.String())
	}
}

// Check refuses a Bare that names none of the tool's verbs. pzm's own
// verbs count as none. It refuses a second Bare too. Such a tool runs
// nothing.
func TestBareIsRefusedWhenItNamesNoVerb(t *testing.T) {
	for _, each := range []struct {
		names []verb.Name
		says  string
	}{
		{[]verb.Name{"glance"}, "pzm.Bare names glance, which is no verb " +
			"of peek's"},
		{[]verb.Name{"seal"}, "pzm.Bare names seal"},
		{[]verb.Name{"look", "tidy"}, "pzm.Bare was called 2 times"},
	} {
		tool := pzm.New("peek", "looks without touching", "", "", "", "")
		tool.Verb("look", verb.NoShort, "says what is there", "# look",
			"Example_look", readsOnly, look)
		tool.Verb("tidy", verb.NoShort, "puts things away", "# tidy", "Example_tidy",
			readsOnly, look)
		for _, name := range each.names {
			tool.Bare(name)
		}
		if err := tool.Check(); err == nil ||
			!strings.Contains(err.Error(), each.says) {
			t.Errorf("Bare %q: Check said %v", each.names, err)
		}
		var out, errs strings.Builder
		if code := tool.Run(nil, &out, &errs); code != exitRefused || out.Len() != none {
			t.Errorf("Bare %q: a bare call ran, code %d, out %q",
				each.names, code, out.String())
		}
	}
}

// Bare may come before the verb it names is declared.
func TestBareMayComeFirst(t *testing.T) {
	tool := pzm.New("peek", "looks without touching", "", "", "", "")
	tool.Bare("look")
	tool.AtTerminal(atATerminal)
	tool.Verb("look", verb.NoShort, "says what is there", "# look", "Example_look",
		readsOnly, look)
	var out, errs strings.Builder
	if code := tool.Run(nil, &out, &errs); code != exitOK ||
		out.String() != "nothing out of place\n" {
		t.Errorf("code %d, out %q, errs %q", code, out.String(),
			errs.String())
	}
}

// doc/bare.md shows the help lines peek prints, word for word.
func TestBareMdShowsWhatTheToolSays(t *testing.T) {
	raw, err := os.ReadFile("../doc/bare.md")
	if err != nil {
		t.Fatal(err)
	}
	page := string(raw)
	for _, each := range []struct {
		line string
		args []string
	}{
		{"peek with nothing after it, at a terminal, is peek look",
			[]string{"--help"}},
		{"peek VERB [ARGS], or peek --VERB [ARGS]", []string{"--help"}},
		{"bare peek: runs look at a terminal, and prints --help elsewhere",
			[]string{"--agent-help"}},
	} {
		if !strings.Contains(page, "\n    "+each.line+"\n") {
			t.Errorf("doc/bare.md does not show %q", each.line)
		}
		var out, errs strings.Builder
		peek(false).Run(each.args, &out, &errs)
		if !strings.Contains(out.String(), each.line+"\n") {
			t.Errorf("peek %q does not say %q:\n%s", each.args, each.line,
				out.String())
		}
	}
}

// asking counts how often the tool asks whether a call is at a terminal.
type asking struct{ times int }

// at counts the question and says yes.
func (count *asking) at(io.Reader, io.Writer) bool {
	count.times++
	return true
}

const (
	// notAsked is how often the tool asks about a call with words. It's
	// defined: never.
	notAsked = 0
	// askedOnce is how often it asks about a call with no words. It's
	// defined: once.
	askedOnce = 1
)

// Bare matters only to a call with no words. So with words the function
// given to AtTerminal is never called.
func TestACallWithWordsIsNotAskedAboutTheTerminal(t *testing.T) {
	for _, words := range [][]string{{"look"}, {"-l"}, {"--help"},
		{"nonsuch"}, {"tidy", "--dry-run"}} {
		count := &asking{}
		tool := peek(false)
		tool.AtTerminal(count.at)
		var out, errs strings.Builder
		tool.Run(words, &out, &errs)
		if count.times != notAsked {
			t.Errorf("%q: asked %d times", words, count.times)
		}
	}
}

// Bare names no verb here. So the tool is not asked either, whether the
// call has words or not.
func TestATerminalIsNotAskedWhenNoVerbIsBare(t *testing.T) {
	for _, words := range [][]string{nil, {}, {"foo-bar"}} {
		count := &asking{}
		tool, _ := built()
		tool.AtTerminal(count.at)
		var out, errs strings.Builder
		tool.Run(words, &out, &errs)
		if count.times != notAsked {
			t.Errorf("%q: asked %d times", words, count.times)
		}
	}
}

// Anti-test: a call with no words is asked once, and a terminal there
// runs the verb Bare named.
func TestACallWithNoWordsIsAskedAboutTheTerminalOnce(t *testing.T) {
	for _, words := range [][]string{nil, {}} {
		count := &asking{}
		tool := peek(false)
		tool.AtTerminal(count.at)
		var out, errs strings.Builder
		tool.Run(words, &out, &errs)
		if count.times != askedOnce ||
			!strings.Contains(out.String(), "nothing out of place") {
			t.Errorf("%q: asked %d times, out %q", words, count.times,
				out.String())
		}
	}
}

// atATerminal says every call is at a terminal. The tests need it, since
// their input and output never are one.
func atATerminal(io.Reader, io.Writer) bool { return true }

// Away from a terminal a bare call prints the help and runs nothing. That
// is how an agent's shell runs it. The verb still runs by name.
func TestABareCallAwayFromATerminalPrintsHelp(t *testing.T) {
	tool := peek(false)
	tool.AtTerminal(func(io.Reader, io.Writer) bool { return false })
	var out, errs strings.Builder
	if code := tool.Run(nil, &out, &errs); code != exitOK ||
		!strings.HasPrefix(out.String(), "peek: looks without touching\n") ||
		strings.Contains(out.String(), "nothing out of place") {
		t.Errorf("peek bare, away from a terminal: code %d, out %q", code,
			out.String())
	}
	out.Reset()
	if code := tool.Run([]string{"look"}, &out, &errs); code != exitOK ||
		out.String() != "nothing out of place\n" {
		t.Errorf("peek look, away from a terminal: code %d, out %q", code,
			out.String())
	}
}

// With no check put in its place, a tool asks of the call itself. None
// of these is a terminal: /dev/null; a pipe; a file on disk; a reader
// that is no file. Each bare call prints the help.
func TestOnlyATerminalIsATerminal(t *testing.T) {
	tool := pzm.New("peek", "looks without touching", "", "", "", "")
	tool.Verb("look", verb.NoShort, "says what is there", "# look", "Example_look",
		readsOnly, look)
	tool.Bare("look")
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	file, err := os.Create(t.TempDir() + "/out")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	for name, in := range map[string]io.Reader{"/dev/null": null,
		"a pipe": reader, "a string": strings.NewReader("")} {
		var out, errs strings.Builder
		tool.RunWith(nil, in, &out, &errs)
		if strings.Contains(out.String(), "nothing out of place") {
			t.Errorf("with %s for input, a bare call ran look", name)
		}
	}
	// As output /dev/null is opened to write; opened to read, writing the
	// help to it fails.
	sink, err := os.Create(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	for name, out := range map[string]*os.File{"/dev/null": sink,
		"a file": file} {
		var errs strings.Builder
		if code := tool.RunWith(nil, null, out, &errs); code != exitOK {
			t.Errorf("with %s for output, a bare call exited %d", name,
				code)
		}
	}
	raw, _ := os.ReadFile(file.Name())
	if strings.Contains(string(raw), "nothing out of place") ||
		!strings.HasPrefix(string(raw), "peek: ") {
		t.Errorf("a bare call written to a file printed %q", raw)
	}
}
