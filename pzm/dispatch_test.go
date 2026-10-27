package pzm_test

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// casesFile lists argument lists, one per line. Each word is a Go
// string. goldenFile holds what the tool says to each, in order.
const (
	casesFile  = "testdata/dispatch-cases.txt"
	goldenFile = "testdata/dispatch-golden.txt"
)

// saying is a verb that writes down that it ran. It gives its name,
// its words and the flags it was given.
type saying struct{ name string }

// do is the verb's run. It writes one line.
func (say saying) do(call pzm.Call) error {
	var line strings.Builder
	fmt.Fprintf(&line, "ran %s words=%q quiet=%t roots=%q", say.name,
		call.Args(), call.On("quiet"), call.Value("roots"))
	if say.name == "tidy" {
		fmt.Fprintf(&line, " dry=%t force=%t", call.DryRun(),
			call.On("force"))
	}
	if say.name == "look" {
		fmt.Fprintf(&line, " from=%q", call.Value("from"))
	}
	_, err := fmt.Fprintln(call.Out(), line.String())
	return err
}

// dispatchTool has a switch and a value of its own.
// Tidy writes. Look reads. Plain has no flags. Boom fails.
func dispatchTool() *pzm.Tool {
	tool := pzm.New("demo", "a tool for testing dispatch", "", "", "", "")
	tool.Flags(pzm.Switch("quiet", 'q', "says less"),
		pzm.Value("roots", 'r', "where the roots are"))
	tool.Verb("tidy", 't', "takes away what is old", "# tidy",
		"ExampleTidy", deletes, saying{"tidy"}.do,
		pzm.Switch("force", 'f', "does not ask"))
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", readsOnly,
		saying{"look"}.do, pzm.Value("from", 'F', "where from"))
	tool.Verb("plain", 'p', "takes no flags", "# plain", "ExamplePlain",
		readsOnly, saying{"plain"}.do)
	tool.Verb("boom", verb.NoShort, "fails", "# boom", "ExampleBoom", readsOnly,
		failing)
	return tool
}

// failing is the verb that fails.
func failing(pzm.Call) error { return fmt.Errorf("boom") }

// wordsOf is the words in a line of Go strings.
func wordsOf(t *testing.T, line string) []string {
	t.Helper()
	words := []string{}
	for rest := strings.TrimSpace(line); rest != ""; {
		quoted, err := strconv.QuotedPrefix(rest)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		word, err := strconv.Unquote(quoted)
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		words = append(words, word)
		rest = strings.TrimSpace(strings.TrimPrefix(rest, quoted))
	}
	return words
}

// said is what the tool does to args, as the golden file has it.
// It gives the arguments, the exit code and what each writer got.
func said(args []string) string { return saidBy(dispatchTool(), args) }

// saidBy is what the tool does to args, as a golden file has it.
func saidBy(tool *pzm.Tool, args []string) string {
	var out, errs bytes.Buffer
	code := tool.Run(args, &out, &errs)
	var quoted []string
	for _, each := range args {
		quoted = append(quoted, strconv.Quote(each))
	}
	return fmt.Sprintf("args %s\ncode %d\nout %q\nerrs %q\n\n",
		strings.Join(quoted, " "), code, out.String(), errs.String())
}

// Every list in the cases file does what the golden file says.
// Reading a call differently shows here as a changed line.
func TestDispatchIsHeldToTheGolden(t *testing.T) {
	cases, err := os.ReadFile(casesFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for _, line := range strings.Split(string(cases), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		args := wordsOf(t, line)
		got.WriteString(said(args))
	}
	if got.String() != string(want) {
		t.Errorf("dispatch changed; diff %s against what this run "+
			"says:\n%s", goldenFile, got.String())
	}
}
