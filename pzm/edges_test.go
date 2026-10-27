package pzm_test

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// lettersOfTheTool are words made only of the letters of the tool's own
// short flags (-q and -r), without a dash. They are words, not flags.
var lettersOfTheTool = []string{"q", "r", "qr", "rq", "qq", "rr"}

// Letters of the tool's short flags with no dash are not flags. They are
// the word where the verb goes, and they name no verb.
func TestAWordOfFlagLettersIsNoVerb(t *testing.T) {
	rapid.Check(t, checkLettersAreNoVerb)
}

// checkLettersAreNoVerb is one draw of that property.
func checkLettersAreNoVerb(t *rapid.T) {
	word := rapid.SampledFrom(lettersOfTheTool).Draw(t, "word")
	args := []string{word, "tidy", "x"}
	code, out, errs := outcome(args)
	want := fmt.Sprintf("demo: there is no verb %s\n", word)
	if code != refusedCode || out != "" || errs != want {
		t.Fatalf("%q: code %d, out %q, errs %q; want %q", args, code, out,
			errs, want)
	}
}

// Anti-test: the same letters with a dash are the tool's flags, and the
// verb after them runs.
func TestFlagLettersWithADashAreFlags(t *testing.T) {
	for _, flags := range [][]string{{"-q"}, {"-r", "x"}, {"-qr", "x"}} {
		args := append(slices.Clone(flags), "plain")
		code, out, errs := outcome(args)
		if code != ranCode || !strings.HasPrefix(out, "ran plain") {
			t.Errorf("%q: code %d, out %q, errs %q", args, code, out, errs)
		}
	}
}

// Alone, a dash is a word and not a flag. Programs read standard input
// when given it. The verb is handed it wherever it comes.
func TestALoneDashIsAWordTheVerbIsGiven(t *testing.T) {
	for _, args := range [][]string{{"-", "tidy"}, {"-q", "-", "tidy"},
		{"tidy", "-"}} {
		code, out, errs := outcome(args)
		if code != ranCode || !strings.HasPrefix(out, "ran tidy words=[\"-\"]") {
			t.Errorf("%q: code %d, out %q, errs %q", args, code, out, errs)
		}
	}
}

// Anti-test: a verb called with no lone dash is handed no such word.
func TestAVerbWithNoLoneDashIsHandedNone(t *testing.T) {
	code, out, errs := outcome([]string{"-q", "tidy"})
	if code != ranCode || !strings.HasPrefix(out, "ran tidy words=[]") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// The switch -q=--quiet takes its long form as its value. It is given. The
// flags after it are still parsed, and the verb asked for by its flag runs.
func TestAShortSwitchGivenItsLongFormLeavesTheRestParsed(t *testing.T) {
	for args, want := range map[string]string{
		"-q=--quiet -l":        "ran look words=[] quiet=true",
		"-q=--quiet --look":    "ran look words=[] quiet=true",
		"-q=--quiet -t -f":     "ran tidy words=[] quiet=true",
		"-q=--quiet look":      "ran look words=[] quiet=true",
		"-q=--quiet -r a -t":   "ran tidy words=[] quiet=true roots=\"a\"",
		"-r a -q=--quiet -p x": "ran plain words=[\"x\"] quiet=true roots=\"a\"",
	} {
		code, out, errs := outcome(strings.Fields(args))
		if code != ranCode || !strings.HasPrefix(out, want) {
			t.Errorf("%s: code %d, out %q, errs %q; want %q", args, code,
				out, errs, want)
		}
	}
}

// Anti-test: a switch given a value that is not its long form is refused,
// and no verb runs.
func TestAShortSwitchGivenAnotherValueIsRefused(t *testing.T) {
	for _, args := range []string{"-q=x -l", "-q=true --look", "-q= look"} {
		code, out, errs := outcome(strings.Fields(args))
		if code != refusedCode || out != "" ||
			!strings.Contains(errs, "a switch takes no value") {
			t.Errorf("%s: code %d, out %q, errs %q", args, code, out, errs)
		}
	}
}

// Commas in the value of a value flag are kept. It is not a list, so it
// has no empty items.
func TestAValueWithCommasIsKeptWhole(t *testing.T) {
	for _, value := range []string{"a,,b", ",", ",a", "a,"} {
		tool, seen := importer()
		code, _, errs := run(tool, "import-agent", "--from", value)
		if code != ranCode || seen.from != value || seen.tags != nil {
			t.Errorf("--from %s: code %d, errs %q, heard %+v", value, code,
				errs, seen)
		}
	}
}

// Anti-test: the same values given to a list are refused for their empty
// item.
func TestTheSameValueGivenToAListIsRefused(t *testing.T) {
	for _, value := range []string{"a,,b", ",", ",a", "a,"} {
		tool, seen := importer()
		code, _, errs := run(tool, "import-agent", "--tags", value)
		if code != refusedCode || !strings.Contains(errs, "has an empty item") ||
			seen.args != nil {
			t.Errorf("--tags %s: code %d, errs %q", value, code, errs)
		}
	}
}

// argsKept are the words the process started with.
type argsKept []string

// restore puts the words back.
func (was argsKept) restore() { os.Args = was }

// startedWith makes the process look as if it started with words. The test
// ends with the real ones back.
func startedWith(t *testing.T, words ...string) {
	t.Helper()
	t.Cleanup(argsKept(os.Args).restore)
	os.Args = words
}

// No words give the same call, nil or empty. Neither reads the words of
// the process. The parser does that when it is given nil.
func TestNoWordsAreNoWordsNilOrEmpty(t *testing.T) {
	startedWith(t, "demo", "plain")
	nilCode, nilOut, nilErrs := outcome(nil)
	code, out, errs := outcome([]string{})
	if nilCode != code || nilOut != out || nilErrs != errs {
		t.Errorf("nil gave %d %q %q; empty gave %d %q %q", nilCode, nilOut,
			nilErrs, code, out, errs)
	}
	if strings.Contains(nilOut, "ran plain") {
		t.Errorf("nil ran the process's words: %q", nilOut)
	}
}

// Anti-test: words given are the words that run, whatever the process was
// started with.
func TestWordsGivenAreNotTheProcessWords(t *testing.T) {
	startedWith(t, "demo", "tidy")
	code, out, errs := outcome([]string{"plain"})
	if code != ranCode || !strings.HasPrefix(out, "ran plain") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}
