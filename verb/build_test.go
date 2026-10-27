package verb_test

import (
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/janearc/puzzlemouth/verb"
	"pgregory.net/rapid"
)

// exampleFile is a test file with the example of the verb declared, which
// holds its output.
const exampleFile = `package foo_test

import "fmt"

func ExampleFooBar() {
	fmt.Println("foo")
	// Output: foo
}
`

// constrained is the example file with a build constraint above it.
func constrained(constraint string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte("//go:build " + constraint +
		"\n\n" + exampleFile)}
}

// plain is the example file as it is.
func plain() *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(exampleFile)}
}

// otherSystems are systems this machine is not, and so builds no file
// named for. Android builds linux files, so linux is not here.
var otherSystems = []string{"aix", "dragonfly", "freebsd", "hurd", "js",
	"netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows"}

// notHere is a system that is not this one.
func notHere() string {
	for _, each := range otherSystems {
		if each != runtime.GOOS {
			return each
		}
	}
	return ""
}

// leftOut is test files that hold foo-bar's example and that go test does
// not compile here, by name.
func leftOut() map[string]*fstest.MapFile {
	return map[string]*fstest.MapFile{
		"foo_test.go":                   constrained("ignore"),
		"bar_test.go":                   constrained("!" + runtime.GOOS),
		"race_test.go":                  constrained("race"),
		"foo_" + notHere() + "_test.go": plain(),
		"_foo_test.go":                  plain(),
		".foo_test.go":                  plain(),
	}
}

// Check and Shown refuse an example that go test does not run because it
// leaves its file out. They name the file and say so.
func TestAnExampleInAFileLeftOutIsRefused(t *testing.T) {
	for name, file := range leftOut() {
		tests := fstest.MapFS{name: file}
		err := verb.Check(tests, declared(t))
		if err == nil || !strings.Contains(err.Error(), name) ||
			!strings.Contains(err.Error(), "leaves out") {
			t.Errorf("%s: Check said %v", name, err)
		}
		if shown, err := verb.Shown(tests, declared(t)); err == nil ||
			!strings.Contains(err.Error(), name) {
			t.Errorf("%s: Shown said %q, %v", name, shown, err)
		}
	}
}

// Anti-test: Check holds an example in a file go test does compile.
func TestAnExampleInAFileThatIsBuiltIsHeld(t *testing.T) {
	for why, tests := range map[string]fstest.MapFS{
		"no constraint": {"foo_test.go": plain()},
		"this system":   {"foo_test.go": constrained(runtime.GOOS)},
		"not ignored":   {"foo_test.go": constrained("!ignore")},
		"its name":      {"foo_" + runtime.GOOS + "_test.go": plain()},
	} {
		if err := verb.Check(tests, declared(t)); err != nil {
			t.Errorf("%s: Check said %v", why, err)
		}
		if _, err := verb.Shown(tests, declared(t)); err != nil {
			t.Errorf("%s: Shown said %v", why, err)
		}
	}
}

// Anti-test: a file left out beside a built one does not hide it. Order
// does not matter, since each system may have its own example.
func TestABuiltExampleBeatsOneLeftOut(t *testing.T) {
	for _, left := range []string{"a_test.go", "z_test.go"} {
		tests := fstest.MapFS{left: constrained("ignore"),
			"m_test.go": plain()}
		if err := verb.Check(tests, declared(t)); err != nil {
			t.Errorf("%s left out: Check said %v", left, err)
		}
	}
}

// Anti-test: a file left out beside one that is also left out stays
// left out, and the first one is the one named.
func TestTwoExamplesLeftOutAreRefused(t *testing.T) {
	tests := fstest.MapFS{"a_test.go": constrained("ignore"),
		"b_test.go": constrained("ignore")}
	err := verb.Check(tests, declared(t))
	if err == nil || !strings.Contains(err.Error(), "a_test.go") {
		t.Errorf("Check said %v", err)
	}
}

// Check and Shown say so when a build constraint cannot be read.
func TestABadConstraintIsSaid(t *testing.T) {
	tests := fstest.MapFS{"bad_test.go": constrained("&& ||")}
	if err := verb.Check(tests, declared(t)); err == nil ||
		!strings.Contains(err.Error(), "bad_test.go") {
		t.Errorf("Check said %v", err)
	}
	if _, err := verb.Shown(tests, declared(t)); err == nil {
		t.Error("Shown showed an example from a file it cannot read")
	}
}

// Check holds the example here exactly when its file is named for this
// system. Any other system leaves it out.
func TestAnExampleIsHeldOnlyOnItsSystem(t *testing.T) {
	rapid.Check(t, checkItsSystem)
}

// checkItsSystem is one draw of that property.
func checkItsSystem(t *rapid.T) {
	systems := append(slices.Clone(otherSystems), runtime.GOOS)
	system := rapid.SampledFrom(systems).Draw(t, "system")
	tests := fstest.MapFS{"foo_" + system + "_test.go": plain()}
	made, err := verb.New("foo-bar", 'f', "foos the bar", "# foo-bar",
		"ExampleFooBar", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	err = verb.Check(tests, made)
	if (err == nil) != (system == runtime.GOOS) {
		t.Fatalf("%s on %s: Check said %v", system, runtime.GOOS, err)
	}
}

// writes are the facts of a verb that writes to the working tree.
var writes = verb.NewFacts(verb.WritesTree, verb.LeavesAlone,
	verb.ThisMachine, verb.NoOne, verb.Light)

// unrelated is an example that gives "--dry-run" to a call that runs
// nothing of the verb's.
var unrelated = fstest.MapFS{"foo_test.go": {Data: []byte(`package foo_test

import "fmt"

func ExampleFooBar() {
	fmt.Println("--dry-run")
	// Output: --dry-run
}
`)}}

// The dry-run check is syntactic, as doc/facts.md says. Any call given
// "--dry-run" satisfies it, even one that runs nothing of the verb's.
// This holds the limit so it is not read as more.
func TestTheDryRunCheckIsOnlySyntactic(t *testing.T) {
	made, err := verb.New("foo-bar", 'f', "foos the bar", "# foo-bar",
		"ExampleFooBar", writes)
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Check(unrelated, made); err != nil {
		t.Errorf("Check refused a syntactic dry run: %v", err)
	}
}

// Anti-test: an example with no "--dry-run" in any call is refused for a
// verb that writes.
func TestAWritingVerbNeedsADryRun(t *testing.T) {
	made, err := verb.New("foo-bar", 'f', "foos the bar", "# foo-bar",
		"ExampleFooBar", writes)
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Check(source, made); err == nil ||
		!strings.Contains(err.Error(), "--dry-run") {
		t.Errorf("Check said %v", err)
	}
}
