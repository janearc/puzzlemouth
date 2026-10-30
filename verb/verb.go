// Package verb is how a command declares what it does. Each verb has one
// line of help and a page for visualdocs. It also has an example that go
// test runs, so the code shown cannot drift from the code that works.
//
// Making a tool not scary is its author's job. The package refuses a
// verb with a part left out when it is made. It refuses a part that
// cannot be what it says too.
package verb

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/janearc/puzzlemouth/config"
)

// Name is a verb's name, lowercase words joined by hyphens: foo-bar. Its
// long flag is the name: --foo-bar.
type Name string

// Short is a verb's one-letter flag, or NoShort. Give a short flag only
// where it earns its letter.
type Short rune

// NoShort is a verb without a short flag.
// It's defined: no letter.
const NoShort Short = 0

// Help is what --help says of a verb for a person. It is one line of
// seventy characters or fewer.
type Help string

// Page is the verb's page in visualdocs. It is markdown drawn by glamour.
type Page string

// Example is the name of the example function that shows a verb at work
// in the command's tests, such as Example_fooBar. go vet refuses an
// example named for an identifier that is not there, so a verb's example
// takes the Example_ form.
type Example string

// Verb is one thing a command does, with all of the above and its facts.
type Verb struct {
	name    Name
	short   Short
	help    Help
	page    Page
	example Example
	facts   Facts
}

// Width is the most characters a line of help may have.
type Width int

// DefaultHelpWidth is the width New and NewFlag hold help to. It is the
// width pzm held help to before it could be set. A line of it stays under
// 80 columns after a two-space margin, so it does not wrap in a default
// terminal when something quotes it.
const DefaultHelpWidth Width = 70

// Unlimited is a width that holds no help to any length.
const Unlimited Width = math.MaxInt

// SetLine reads a width from the line of a config: one word, a positive
// whole number. A word that is not one is refused, and the width is left
// as it was. It is what config reads a help-width line with.
func (width *Width) SetLine(line config.Line) error {
	word := line.Words[firstWord]
	got, err := strconv.Atoi(word)
	if err != nil || got <= none {
		return fmt.Errorf("%q is not a positive whole number", word)
	}
	*width = Width(got)
	return nil
}

// TooLong is why help is over width characters, or nothing when it is
// not. Characters are counted, not bytes.
func TooLong(help Help, width Width) error {
	if utf8.RuneCountInString(string(help)) <= int(width) {
		return nil
	}
	return fmt.Errorf("help: it is over %d characters", width)
}

// named is what a verb's name may be.
var named = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// New is a verb, or why it cannot be one. It gives every refusal at once.
// Each names the part it is about.
func New(name Name, short Short, help Help, page Page,
	example Example, facts Facts) (*Verb, error) {
	return NewWithin(DefaultHelpWidth, name, short, help, page, example,
		facts)
}

// NewWithin is New with help held to width rather than the default.
func NewWithin(width Width, name Name, short Short, help Help, page Page,
	example Example, facts Facts) (*Verb, error) {
	refused := partsRefused(width, name, short, help)
	refuse := refused.refuse
	pageText, exampleText := string(page), string(example)
	refuse(strings.TrimSpace(pageText) != "", "page", "it is empty")
	namesOne := strings.HasPrefix(exampleText, "Example") &&
		len(example) > len("Example")
	notAnExample := fmt.Sprintf(
		"%q does not name an example function", example)
	refuse(namesOne, "example", notAnExample)
	for _, why := range facts.refusals() {
		refuse(false, "facts", why)
	}
	if len(refused) > none {
		return nil, fmt.Errorf("verb %s: %w", name,
			errors.Join(refused...))
	}
	return &Verb{name, short, help, page, example, facts}, nil
}

// refusals are what is wrong with a verb or a flag. Each names its part.
type refusals []error

// refuse keeps why under the part when sound is false.
func (refused *refusals) refuse(sound bool, part, why string) {
	if !sound {
		*refused = append(*refused, errors.New(part+": "+why))
	}
}

// partsRefused is what is wrong with the name, short flag and help that
// a verb and a flag share. They come in that order.
func partsRefused(width Width, name Name, short Short, help Help) refusals {
	var refused refusals
	refuse := refused.refuse
	nameText, helpText := string(name), string(help)
	notNamed := fmt.Sprintf(
		"%q is not lowercase words joined by hyphens", name)
	letter := rune(short)
	notALetter := fmt.Sprintf("%q is not a letter or a digit", letter)
	refuse(named.MatchString(nameText), "name", notNamed)
	refuse(short == NoShort || isLetter(short), "short", notALetter)
	refuse(oneLine(helpText), "help", notOneLine)
	if err := TooLong(help, width); err != nil {
		refused = append(refused, err)
	}
	return refused
}

// notOneLine is why a line of help is refused.
const notOneLine = "it is empty or more than one line"

// letters are what a short flag may be: a shell passes each as it is.
const letters = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// isLetter is whether a short flag is one of the letters.
func isLetter(short Short) bool {
	return short != NoShort && strings.ContainsRune(letters, rune(short))
}

// oneLine is whether text is one line with something on it.
func oneLine(text string) bool {
	return strings.TrimSpace(text) != "" &&
		!strings.ContainsAny(text, "\r\n")
}

// Name is the verb's name.
func (made Verb) Name() string { return string(made.name) }

// Long is the verb's long flag, its name after two hyphens.
func (made Verb) Long() string { return "--" + string(made.name) }

// Short is the verb's short flag after one hyphen, or nothing when it has
// none.
func (made Verb) Short() string {
	if made.short == NoShort {
		return ""
	}
	return "-" + string(rune(made.short))
}

// Help is the verb's line for --help.
func (made Verb) Help() string { return string(made.help) }

// Page is the verb's page in visualdocs.
func (made Verb) Page() string { return string(made.page) }

// Example is the name of the verb's example function.
func (made Verb) Example() string { return string(made.example) }

// Facts are what the verb does to the machine it runs on.
func (made Verb) Facts() Facts { return made.facts }
