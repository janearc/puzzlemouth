package pzm_test

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// readsOnly are the facts of a verb that only reads, as most of the
// tests' verbs do.
var readsOnly = verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
	verb.ThisMachine, verb.NoOne, verb.Light)

// ran is what a test tool's verbs saw: which verb, and its arguments.
type ran struct{ said []string }

// fooBar is the run of foo-bar: it says its arguments.
func (seen *ran) fooBar(call pzm.Call) error {
	seen.said = append(seen.said, "foo-bar "+strings.Join(call.Args(), " "))
	_, err := call.Out().Write([]byte("foo " + strings.Join(call.Args(), " ")))
	return err
}

// built is a tool with two verbs, foo-bar and baz, and what they saw.
func built() (*pzm.Tool, *ran) {
	seen := &ran{}
	tool := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	tool.Verb("foo-bar", 'f', "foos the bar", "# foo-bar", "ExampleFooBar", readsOnly,
		seen.fooBar)
	tool.Verb("baz", 'b', "fails, always", "# baz", "ExampleBaz", readsOnly,
		func(pzm.Call) error { return errors.New("boom") })
	return tool, seen
}

// run is a tool run with args: its exit code, and what it wrote to each.
func run(tool *pzm.Tool, args ...string) (int, string, string) {
	var out, errs bytes.Buffer
	code := tool.Run(args, &out, &errs)
	return code, out.String(), errs.String()
}

// A verb runs by its name, with the arguments after it.
func TestAVerbRunsByName(t *testing.T) {
	tool, seen := built()
	code, out, _ := run(tool, "foo-bar", "x", "y")
	if code != exitOK || out != "foo x y" || len(seen.said) != one {
		t.Errorf("code %d, out %q, saw %v", code, out, seen.said)
	}
}

// A verb answers to its long flag and its short one as to its name.
func TestAVerbAnswersToItsFlags(t *testing.T) {
	for _, asked := range []string{"--foo-bar", "-f"} {
		tool, _ := built()
		code, out, errs := run(tool, asked, "x")
		if code != exitOK || out != "foo x" {
			t.Errorf("%s: code %d, out %q, errs %q", asked, code, out, errs)
		}
	}
}

// A verb that fails says why, under the tool's and the verb's name, and
// the tool exits 1.
func TestAFailingVerbSaysWhy(t *testing.T) {
	tool, _ := built()
	code, _, errs := run(tool, "baz")
	if code != exitFailed || !strings.Contains(errs, "demo baz: boom") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// A tool asked for what it does not have says so and exits 2, and runs
// nothing.
func TestWhatIsNotThereExitsTwo(t *testing.T) {
	for _, asked := range [][]string{{"nope"}, {"--nope"},
		{"--foo-bar", "--baz"}} {
		tool, seen := built()
		code, _, errs := run(tool, asked...)
		if code != exitRefused || errs == "" || len(seen.said) != none {
			t.Errorf("%v: code %d, errs %q, ran %v", asked, code, errs, seen.said)
		}
	}
}

// A tool whose verbs are declared wrong never runs: it says what is wrong
// and exits 2, and Check, for its tests, says the same.
func TestABadDeclarationNeverRuns(t *testing.T) {
	tool, seen := built()
	tool.Verb("Bad Name", 'q', "h", "p", "ExampleBad", readsOnly, seen.fooBar)
	code, _, errs := run(tool, "foo-bar")
	if code != exitRefused || !strings.Contains(errs, "Bad Name") || len(seen.said) != none {
		t.Errorf("code %d, errs %q, ran %v", code, errs, seen.said)
	}
	if err := tool.Check(); err == nil || !strings.Contains(err.Error(), "Bad Name") {
		t.Errorf("Check said %v", err)
	}
}

// Examples holds a tool's verbs to examples that are there and print what
// they claim, so the docs cannot drift. Nothing in pzm
// called verb.Check.
func TestExamplesAreHeld(t *testing.T) {
	nothing := func(pzm.Call) error { return nil }
	held := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	held.Verb("make", 'm', "makes a tool", "# make", "ExampleNew", readsOnly, nothing)
	if err := held.Examples(os.DirFS(".")); err != nil {
		t.Errorf("an example that is there and holds: %v", err)
	}
	missing := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	missing.Verb("gone", 'g', "has no example", "# gone", "ExampleGone", readsOnly,
		nothing)
	if missing.Examples(os.DirFS(".")) == nil {
		t.Error("a verb with no example was let through")
	}
}

// A verb declared with nothing to do is refused like any other part left
// out, so the tool never calls through nil. It was found by probing.
func TestAVerbWithNothingToDoIsRefused(t *testing.T) {
	tool, seen := built()
	tool.Verb("idle", 'i', "does nothing", "# idle", "ExampleIdle", readsOnly, nil)
	code, _, errs := run(tool, "idle")
	if code != exitRefused || !strings.Contains(errs, "give it a Do") || len(seen.said) != none {
		t.Errorf("code %d, errs %q, ran %v", code, errs, seen.said)
	}
	if tool.Check() == nil {
		t.Error("Check let a verb with no Do in")
	}
}

// -h is pzm's, for help. A verb given it is refused when declared, not
// left for cobra to panic over. Found by probing, 2026-10-15.
func TestShortHIsPzms(t *testing.T) {
	tool, _ := built()
	tool.Verb("hello", 'h', "says hello", "# hello", "ExampleHello", readsOnly,
		func(pzm.Call) error { return nil })
	code, _, errs := run(tool, "--help")
	if code != exitRefused || !strings.Contains(errs, "-h is pzm's own") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// Two verbs may not share a name or a short flag, and any number may go
// without a short flag.
func TestTwoVerbsCannotBeOne(t *testing.T) {
	nothing := func(pzm.Call) error { return nil }
	for _, second := range []struct {
		why   string
		name  verb.Name
		short verb.Short
	}{{"the same name", "foo-bar", 'q'}, {"the same short flag", "other", 'f'}} {
		tool, _ := built()
		tool.Verb(second.name, second.short, "h", "p", "ExampleOther", readsOnly, nothing)
		if tool.Check() == nil {
			t.Errorf("%s was let in", second.why)
		}
	}
	tool := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	tool.Verb("one", verb.NoShort, "h", "p", "ExampleOne", readsOnly, nothing)
	tool.Verb("two", verb.NoShort, "h", "p", "ExampleTwo", readsOnly, nothing)
	if err := tool.Check(); err != nil {
		t.Errorf("two verbs without short flags: %v", err)
	}
}

// --help, or nothing at all, lists every verb with its line of help.
func TestHelpListsEveryVerb(t *testing.T) {
	for _, asked := range [][]string{{"--help"}, {}} {
		tool, _ := built()
		code, out, _ := run(tool, asked...)
		for _, want := range []string{"demo", "foo-bar", "foos the bar",
			"baz", "fails, always"} {
			if !strings.Contains(out, want) {
				t.Errorf("%v: code %d, help has no %q:\n%s", asked, code, want, out)
			}
		}
	}
}

// Nothing pzm exports names cobra or pflag, so a tool built on pzm cannot
// reach through it to the parser underneath.
func TestNothingExportedNamesTheParserUnderneath(t *testing.T) {
	places := token.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(places, name, raw, none)
		if err != nil {
			t.Fatal(err)
		}
		for _, declared := range file.Decls {
			if said := exported(places, declared); strings.Contains(said, "cobra") ||
				strings.Contains(said, "pflag") {
				t.Errorf("%s exports: %s", name, said)
			}
		}
	}
}

// exported is how an exported declaration reads, without bodies, or
// nothing when it is not exported.
func exported(places *token.FileSet, declared ast.Decl) string {
	var said bytes.Buffer
	switch each := declared.(type) {
	case *ast.FuncDecl:
		if !each.Name.IsExported() {
			return ""
		}
		each.Body = nil
		printer.Fprint(&said, places, each)
	case *ast.GenDecl:
		for _, spec := range each.Specs {
			if named, isType := spec.(*ast.TypeSpec); isType &&
				named.Name.IsExported() {
				printer.Fprint(&said, places, named)
			}
		}
	}
	return said.String()
}
