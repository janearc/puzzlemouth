package verb_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/janearc/puzzlemouth/verb"
)

// readsOnly are the facts of a verb that only reads, as most of the
// tests' verbs do.
var readsOnly = verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
	verb.ThisMachine, verb.NoOne, verb.Light)

// declared is a verb whose every part is sound, for tests to change one
// part of at a time.
func declared(t *testing.T) *verb.Verb {
	t.Helper()
	made, err := verb.New("foo-bar", 'f', "foos the bar",
		"# foo-bar\n\nfoos the bar.", "ExampleFooBar", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// A verb says back every part it was declared with, and its long flag is
// its name.
func TestAVerbSaysWhatItWasDeclaredWith(t *testing.T) {
	made := declared(t)
	if made.Name() != "foo-bar" || made.Long() != "--foo-bar" ||
		made.Short() != "-f" || made.Help() != "foos the bar" ||
		made.Example() != "ExampleFooBar" {
		t.Errorf("declared foo-bar, got %q %q %q %q %q", made.Name(),
			made.Long(), made.Short(), made.Help(), made.Example())
	}
	if !strings.HasPrefix(made.Page(), "# foo-bar") {
		t.Errorf("page %q", made.Page())
	}
}

// A verb with no short flag has none to say.
func TestAShortFlagIsOptional(t *testing.T) {
	made, err := verb.New("age", verb.NoShort, "which commit this is",
		"# age", "ExampleAge", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if made.Short() != "" {
		t.Errorf("a verb with no short flag says %q", made.Short())
	}
}

// New refuses every part that cannot be what it says, and says which.
func TestNewRefusesWhatCannotBe(t *testing.T) {
	long := strings.Repeat("x", tooWide)
	cases := []struct {
		why     string
		name    verb.Name
		short   verb.Short
		help    verb.Help
		page    verb.Page
		example verb.Example
		says    string
	}{
		{"a name with capitals", "Foo-bar", 'f', "h", "p", "ExampleFoo", "name"},
		{"a name with a space", "foo bar", 'f', "h", "p", "ExampleFoo", "name"},
		{"a name ending in a hyphen", "foo-", 'f', "h", "p", "ExampleFoo", "name"},
		{"no name", "", 'f', "h", "p", "ExampleFoo", "name"},
		{"a short flag that is not a letter", "foo", '-', "h", "p", "ExampleFoo", "short"},
		{"no help", "foo", 'f', "", "p", "ExampleFoo", "help"},
		{"help of 71 characters", "foo", 'f', verb.Help(long), "p", "ExampleFoo", "help"},
		{"help on two lines", "foo", 'f', "h\nh", "p", "ExampleFoo", "help"},
		{"no page", "foo", 'f', "h", "", "ExampleFoo", "page"},
		{"an example not named Example", "foo", 'f', "h", "p", "TestFoo", "example"},
		{"an example named only Example", "foo", 'f', "h", "p", "Example", "example"},
	}
	for _, each := range cases {
		_, err := verb.New(each.name, each.short, each.help, each.page,
			each.example, readsOnly)
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("%s: New said %v, want a refusal naming the %s",
				each.why, err, each.says)
		}
	}
}

// A short flag is a letter or a digit, any of them, and nothing beside
// them; the zero Short is no short flag at all.
func TestShortFlagsAreLettersAndDigits(t *testing.T) {
	for _, short := range "azAZ09" {
		made, err := verb.New("foo", verb.Short(short), "h", "p", "ExampleFoo", readsOnly)
		if err != nil || made.Short() != "-"+string(short) {
			t.Errorf("-%c: %q, %v", short, made.Short(), err)
		}
	}
	for _, short := range "`{@[/:é" {
		if _, err := verb.New("foo", verb.Short(short), "h", "p",
			"ExampleFoo", readsOnly); err == nil {
			t.Errorf("-%c was let in", short)
		}
	}
	made, err := verb.New("foo", verb.NoShort, "h", "p", "ExampleFoo", readsOnly)
	if err != nil || made.Short() != "" {
		t.Errorf("the zero Short: %q, %v", made.Short(), err)
	}
}

// Help of exactly seventy characters is allowed; the count is in
// characters, not bytes.
func TestHelpIsCountedInCharacters(t *testing.T) {
	help := verb.Help(strings.Repeat("é", widest))
	if _, err := verb.New("foo", 'f', help, "p", "ExampleFoo", readsOnly); err != nil {
		t.Errorf("seventy characters were refused: %v", err)
	}
}

// source is a command's test file with one example that holds its
// output, one that does not, and a function that is not an example.
var source = fstest.MapFS{"foo_test.go": {Data: []byte(`package foo_test

import "fmt"

func ExampleFooBar() {
	fmt.Println("foo")
	// Output: foo
}

func ExampleUnheld() {
	fmt.Println("foo")
}

func ExampleWithArgs(each int) {
	fmt.Println(each)
	// Output: 1
}

func helper() {}
`)}}

// broken is a command's tests where one file does not parse, beside the
// file that holds foo-bar's example.
var broken = fstest.MapFS{
	"foo_test.go":    source["foo_test.go"],
	"broken_test.go": {Data: []byte("package foo_test\n\nfunc {")},
}

// Check and Shown say so when a test file does not parse, rather than
// answering from the files that did.
func TestABrokenTestFileIsSaid(t *testing.T) {
	if err := verb.Check(broken, declared(t)); err == nil ||
		!strings.Contains(err.Error(), "broken_test.go") {
		t.Errorf("Check said %v", err)
	}
	if _, err := verb.Shown(broken, declared(t)); err == nil ||
		!strings.Contains(err.Error(), "broken_test.go") {
		t.Errorf("Shown said %v", err)
	}
}

// errUnread is what reading gone_test.go says.
var errUnread = errors.New("gone_test.go cannot be read")

// unread is a command's tests that list a file they cannot read.
type unread struct{ fstest.MapFS }

// ReadFile reads every file but gone_test.go.
func (tests unread) ReadFile(name string) ([]byte, error) {
	if name == "gone_test.go" {
		return nil, errUnread
	}
	return tests.MapFS.ReadFile(name)
}

// Check and Shown pass on the error when a test file cannot be read.
func TestAnUnreadTestFileIsSaid(t *testing.T) {
	tests := unread{fstest.MapFS{"foo_test.go": source["foo_test.go"],
		"gone_test.go": {Data: []byte("package foo_test")}}}
	if err := verb.Check(tests, declared(t)); !errors.Is(err, errUnread) {
		t.Errorf("Check said %v", err)
	}
	if _, err := verb.Shown(tests, declared(t)); !errors.Is(err, errUnread) {
		t.Errorf("Shown said %v", err)
	}
}

// A function named like an example that go test would not run, because
// it takes an argument, is not an example.
func TestOnlyRealExamplesCount(t *testing.T) {
	made, err := verb.New("foo", 'f', "h", "p", "ExampleWithArgs", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if err := verb.Check(source, made); err == nil ||
		!strings.Contains(err.Error(), "not in the tests") {
		t.Errorf("Check said %v", err)
	}
	if shown, err := verb.Shown(source, made); err == nil {
		t.Errorf("Shown showed %q", shown)
	}
}

// Check passes a verb whose example is in the source and holds its
// output, so go test runs it and compares what it prints.
func TestCheckPassesAHeldExample(t *testing.T) {
	if err := verb.Check(source, declared(t)); err != nil {
		t.Error(err)
	}
}

// Check refuses a verb whose example is missing, or prints without an
// Output comment, which go test would compile but never compare.
func TestCheckRefusesAnExampleThatCanDrift(t *testing.T) {
	for name, says := range map[verb.Example]string{
		"ExampleMissing": "not in the tests", "ExampleUnheld": "no Output"} {
		made, err := verb.New("foo", 'f', "h", "p", name, readsOnly)
		if err != nil {
			t.Fatal(err)
		}
		if err := verb.Check(source, made); err == nil ||
			!strings.Contains(err.Error(), string(name)) ||
			!strings.Contains(err.Error(), says) {
			t.Errorf("%s: Check said %v, want %q", name, err, says)
		}
	}
	missing, err := verb.New("foo", 'f', "h", "p", "ExampleMissing", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if shown, err := verb.Shown(source, missing); err == nil {
		t.Errorf("Shown showed a missing example as %q", shown)
	}
}

// Check refuses two verbs with one name, or one short flag.
func TestCheckRefusesTwoOfOne(t *testing.T) {
	again, err := verb.New("foo-bar", 'g', "h", "p", "ExampleFooBar", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	other, err := verb.New("other", 'f', "h", "p", "ExampleFooBar", readsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, second := range []*verb.Verb{again, other} {
		if err := verb.Check(source, declared(t), second); err == nil {
			t.Errorf("Check let %s beside foo-bar", second.Name())
		}
	}
}

// Shown is the example as it is written, the whole function, for the
// code window in visualdocs.
func TestShownIsTheExampleAsWritten(t *testing.T) {
	shown, err := verb.Shown(source, declared(t))
	if err != nil {
		t.Fatal(err)
	}
	want := "func ExampleFooBar() {\n\tfmt.Println(\"foo\")\n\t// Output: foo\n}"
	if shown != want {
		t.Errorf("shown:\n%s\nwant:\n%s", shown, want)
	}
}
