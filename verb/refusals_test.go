package verb_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/verb"
)

// Every part of a verb wrong gives every refusal at once. They come in
// the order of the parts, and flags share the first three.
func TestAVerbWrongInEveryPartSaysSoInOrder(t *testing.T) {
	_, err := verb.New("Foo Bar", '!', "", " ", "foo", verb.Facts{})
	if err == nil {
		t.Fatal("a verb wrong in every part was made")
	}
	want := strings.Join([]string{
		`verb Foo Bar: name: "Foo Bar" is not lowercase words joined by hyphens`,
		`short: '!' is not a letter or a digit`,
		`help: it is empty or more than one line`,
		`page: it is empty`,
		`example: "foo" does not name an example function`,
	}, "\n")
	if !strings.HasPrefix(err.Error(), want) {
		t.Errorf("got:\n%s\nwant it to start:\n%s", err, want)
	}
}

// Every part of a flag wrong gives the same refusals for the three parts
// it has, in the same order and words.
func TestAFlagWrongInEveryPartSaysSoInOrder(t *testing.T) {
	_, err := verb.NewFlag(verb.DefaultHelpWidth, "Foo Bar", '!', "")
	if err == nil {
		t.Fatal("a flag wrong in every part was made")
	}
	want := strings.Join([]string{
		`flag --Foo Bar: name: "Foo Bar" is not lowercase words joined by hyphens`,
		`short: '!' is not a letter or a digit`,
		`help: it is empty or more than one line`,
	}, "\n")
	if err.Error() != want {
		t.Errorf("got:\n%s\nwant:\n%s", err, want)
	}
}

const (
	// widest is how many characters help may have. It's defined by
	// verb.New: 70.
	widest = 70
	// tooWide is one more. It's defined.
	tooWide = widest + 1
)

// Both refuse help over the width in the same words. They count
// characters and not bytes, so the widest help is held.
func TestHelpOverTheWidthIsRefusedByBoth(t *testing.T) {
	over := strings.Repeat("é", tooWide)
	_, flagErr := verb.NewFlag(verb.DefaultHelpWidth, "foo", verb.NoShort, verb.Help(over))
	_, verbErr := verb.New("foo", verb.NoShort, verb.Help(over), "# p",
		"ExampleFoo", readsOnly)
	for who, err := range map[string]error{"flag": flagErr, "verb": verbErr} {
		if err == nil || !strings.Contains(err.Error(),
			"help: it is over 70 characters") {
			t.Errorf("%s: %v", who, err)
		}
	}
	exact := strings.Repeat("é", widest)
	if _, err := verb.NewFlag(verb.DefaultHelpWidth, "foo", verb.NoShort, verb.Help(exact)); err != nil {
		t.Errorf("the widest help was refused: %v", err)
	}
}
