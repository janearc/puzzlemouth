package verb_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/verb"
)

const (
	// seventy is the default width. It's defined by verb.DefaultHelpWidth.
	seventy = 70
	// ten is a width a test sets. It's defined here.
	ten = 10
	// three is a narrower width a test sets. It's defined here.
	three = 3
	// wayOver is a length over any width a test sets. It's defined here.
	wayOver = 500
	// one is a count of one. It's defined here.
	one = 1
)

// The default width is seventy characters.
func TestTheDefaultHelpWidthIsSeventy(t *testing.T) {
	if int(verb.DefaultHelpWidth) != seventy {
		t.Errorf("got %d", verb.DefaultHelpWidth)
	}
}

// A verb and a flag made within a width accept help of that width and
// refuse one character more, in the words New and NewFlag use.
func TestAVerbAndAFlagMadeWithinAWidthHoldToIt(t *testing.T) {
	const width = ten
	fits, over := strings.Repeat("é", width), strings.Repeat("é", width+one)
	_, err := verb.NewWithin(width, "foo", verb.NoShort, verb.Help(fits),
		"# p", "ExampleFoo", readsOnly)
	if err != nil {
		t.Errorf("a verb of exactly the width: %v", err)
	}
	_, err = verb.NewFlag(width, "foo", verb.NoShort, verb.Help(fits))
	if err != nil {
		t.Errorf("a flag of exactly the width: %v", err)
	}
	_, err = verb.NewWithin(width, "foo", verb.NoShort, verb.Help(over),
		"# p", "ExampleFoo", readsOnly)
	if err == nil || !strings.Contains(err.Error(), "over 10 characters") {
		t.Errorf("a verb one over: %v", err)
	}
	_, err = verb.NewFlag(width, "foo", verb.NoShort, verb.Help(over))
	if err == nil || !strings.Contains(err.Error(), "over 10 characters") {
		t.Errorf("a flag one over: %v", err)
	}
}

// Anti-test: Unlimited holds no help to a width, and New still holds it
// to the default.
func TestUnlimitedHoldsNothingAndNewHoldsTheDefault(t *testing.T) {
	long := verb.Help(strings.Repeat("x", wayOver))
	if _, err := verb.NewFlag(verb.Unlimited, "foo", verb.NoShort,
		long); err != nil {
		t.Errorf("unlimited: %v", err)
	}
	if _, err := verb.NewFlag(verb.DefaultHelpWidth, "foo", verb.NoShort, long); err == nil {
		t.Error("NewFlag took help over the default")
	}
}

// TooLong says what is wrong with help over a width, and nothing for
// help within it.
func TestTooLongNamesHelpOverTheWidth(t *testing.T) {
	if err := verb.TooLong("abc", three); err != nil {
		t.Errorf("at the width: %v", err)
	}
	err := verb.TooLong("abcd", three)
	if err == nil || err.Error() != "help: it is over 3 characters" {
		t.Errorf("over the width: %v", err)
	}
}
