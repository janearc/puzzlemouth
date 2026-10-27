package verb_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/verb"
)

// A flag says back what it was declared with; its long form is its name.
func TestAFlagSaysWhatItWasDeclaredWith(t *testing.T) {
	flag, err := verb.NewFlag(verb.DefaultHelpWidth, "dry-run", 'n', "says what it would do")
	if err != nil {
		t.Fatal(err)
	}
	if flag.Name() != "dry-run" || flag.Long() != "--dry-run" ||
		flag.Short() != "-n" || flag.Help() != "says what it would do" {
		t.Errorf("got %q %q %q %q", flag.Name(), flag.Long(), flag.Short(),
			flag.Help())
	}
	plain, err := verb.NewFlag(verb.DefaultHelpWidth, "hook", verb.NoShort, "read the payload")
	if err != nil || plain.Short() != "" {
		t.Errorf("no short flag: %q %v", plain.Short(), err)
	}
}

// A flag is held to a verb's rules, and every part that fails is named
// at once.
func TestAFlagIsHeldToAVerbsRules(t *testing.T) {
	_, err := verb.NewFlag(verb.DefaultHelpWidth, "Dry Run", '!', "")
	if err == nil {
		t.Fatal("a flag with every part wrong was made")
	}
	for _, part := range []string{"flag --Dry Run", "name:", "short:",
		"help:"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("no %q in %v", part, err)
		}
	}
	long := strings.Repeat("x", tooWide)
	if _, err := verb.NewFlag(verb.DefaultHelpWidth, "wide", verb.NoShort, verb.Help(long)); err == nil ||
		!strings.Contains(err.Error(), "over 70") {
		t.Errorf("a line of 71: %v", err)
	}
	if _, err := verb.NewFlag(verb.DefaultHelpWidth, "edge", verb.NoShort, verb.Help(long[:widest])); err != nil {
		t.Errorf("a line of 70: %v", err)
	}
	if _, err := verb.NewFlag(verb.DefaultHelpWidth, "two", verb.NoShort, "one\ntwo"); err == nil {
		t.Error("two lines of help")
	}
}
