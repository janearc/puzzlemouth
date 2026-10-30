package verb_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
	"github.com/janearc/puzzlemouth/verb"
)

// widthFile is a config of one help-width line, as pzm's own has it.
type widthFile struct {
	Width verb.Width `config:"help-width,1" help:"the most characters a line of help may have"`
}

// readWidth is the text read as a config of a help-width, starting from
// the default.
func readWidth(t *testing.T, text string) (widthFile, error) {
	t.Helper()
	schema, err := config.NewSchema[widthFile]()
	if err != nil {
		t.Fatal(err)
	}
	got := widthFile{Width: verb.DefaultHelpWidth}
	err = schema.Read("config", []byte(text), &got)
	return got, err
}

// A help-width line sets the width to its one word.
func TestAWidthIsReadFromItsLine(t *testing.T) {
	got, err := readWidth(t, "help-width 90\n")
	if err != nil || got.Width != 90 {
		t.Errorf("got %d, %v", got.Width, err)
	}
}

// Anti-test: a word that is not a positive whole number is refused,
// naming the word, and the width stays as it was.
func TestAWidthThatIsNotPositiveIsRefusedAndKeptAsItWas(t *testing.T) {
	for _, word := range []string{"0", "-3", "7.5", "wide",
		"99999999999999999999"} {
		got, err := readWidth(t, "help-width "+word+"\n")
		if err == nil || !strings.Contains(err.Error(), word) {
			t.Errorf("%s: %v", word, err)
		}
		if got.Width != verb.DefaultHelpWidth {
			t.Errorf("%s changed the width to %d", word, got.Width)
		}
	}
}

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
