package verb_test

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/janearc/puzzlemouth/verb"
	"pgregory.net/rapid"
)

// lowercaseWords is what a name may be, as doc/verbs.md says it:
// lowercase words joined by hyphens.
var lowercaseWords = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// nameChars are the characters a drawn name is made of. Most are
// allowed in a name somewhere and not everywhere.
const nameChars = "ab9-_ AZ."

// shortChars are the letters and digits a short flag may be.
const shortChars = "azAZ09"

// helpChars make the drawn help. Some are letters. One is a space, one a
// newline and one a carriage return. Two take two bytes.
const helpChars = "ab \n\réü"

const (
	// fewest is the fewest characters a drawn name or help has. It's
	// defined.
	fewest = 0
	// longest is the most. It's defined.
	longest = 90
)

// drawn is a string of the characters, as long as longest allows.
func drawn(t *rapid.T, chars, label string) string {
	char := rapid.SampledFrom([]rune(chars))
	return rapid.StringOfN(char, fewest, longest, longest).Draw(t, label)
}

// declaredWith is a verb made from the parts, with an example and page
// that are sound.
func declaredWith(name string, short verb.Short, help string) (verb.Verb, error) {
	return verb.New(verb.Name(name), short, verb.Help(help),
		"# a page", "ExampleDrawn", readsOnly)
}

// Sound parts come back as they were declared. So do the facts.
func TestAVerbIsReadBackAsDeclared(t *testing.T) {
	rapid.Check(t, checkReadBack)
}

// checkReadBack is one draw of that property.
func checkReadBack(t *rapid.T) {
	name := rapid.StringMatching(`[a-z][a-z0-9]{0,6}(-[a-z0-9]{1,4}){0,2}`).
		Draw(t, "name")
	short := rapid.SampledFrom([]rune(shortChars)).Draw(t, "short")
	help := rapid.StringMatching(`[a-z][a-z ]{0,30}[a-z]`).Draw(t, "help")
	made, err := declaredWith(name, verb.Short(short), help)
	if err != nil {
		t.Fatal(err)
	}
	if made.Name() != name || made.Long() != "--"+name ||
		made.Short() != "-"+string(short) || made.Help() != help ||
		made.Facts() != readsOnly {
		t.Fatalf("%q %q %q came back as %+v", name, string(short), help, made)
	}
}

// Anti-test: a name is refused exactly when it is not lowercase words
// joined by hyphens, whatever the characters.
func TestANameIsKeptOnlyIfItIsLowercaseWords(t *testing.T) {
	rapid.Check(t, checkNameRule)
}

// checkNameRule is one draw of that property.
func checkNameRule(t *rapid.T) {
	name := drawn(t, nameChars, "name")
	_, err := declaredWith(name, verb.NoShort, "helps")
	if (err == nil) != lowercaseWords.MatchString(name) {
		t.Fatalf("%q: %v", name, err)
	}
	if err != nil && !strings.Contains(err.Error(), "name:") {
		t.Fatalf("%q: the refusal does not name the part: %v", name, err)
	}
}

// Anti-test: help is refused exactly when it is blank, has a line break
// or is over seventy characters. Characters are counted, not bytes.
func TestHelpIsKeptOnlyIfItIsOneShortLine(t *testing.T) {
	rapid.Check(t, checkHelpRule)
}

// checkHelpRule is one draw of that property.
func checkHelpRule(t *rapid.T) {
	help := drawn(t, helpChars, "help")
	_, err := declaredWith("helped", verb.NoShort, help)
	sound := strings.TrimSpace(help) != "" &&
		!strings.ContainsAny(help, "\r\n") &&
		utf8.RuneCountInString(help) <= widest
	if (err == nil) != sound {
		t.Fatalf("%q: sound %v, %v", help, sound, err)
	}
}
