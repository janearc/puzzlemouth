package numbers_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// sourceLines are the lines of sound's measured number that make up its
// source, so a test can take one away or all.
var sourceLines = map[string]string{
	"key":   "      key: nbsir-79-1910\n",
	"where": "      where: table 1, pdf page 35, row v/z^1/2, weighted average\n",
	"page":  "      page: 35\n",
	"text":  "      text: \"6.84\"\n",
}

// once is how many times a part is taken out of a file. It's defined.
const once = 1

// without is text with the first of each part taken out.
func without(text string, parts ...string) string {
	for _, part := range parts {
		text = strings.Replace(text, part, "", once)
	}
	return text
}

// Anti-test: Read refuses a measured number with no source. It refuses one
// with no key, and one with no where. The refusal names the number.
func TestAMeasuredNumberMissingItsSourcePartIsRefused(t *testing.T) {
	whole := sourceLines["key"] + sourceLines["where"] +
		sourceLines["page"] + sourceLines["text"]
	for why, file := range map[string]string{
		"no source":           without(sound, "    source:\n"+whole),
		"no key":              without(sound, sourceLines["key"]),
		"no where":            without(sound, sourceLines["where"]),
		"no key and no where": without(sound, sourceLines["key"], sourceLines["where"]),
	} {
		if file == sound {
			t.Fatalf("%s: the test did not change the file", why)
		}
		_, err := numbers.Read("bad.yaml", []byte(file))
		if err == nil || !strings.Contains(err.Error(),
			"continuous-speed: a measured number names") {
			t.Errorf("%s: %v", why, err)
		}
	}
}

// The page of a source is not needed: a measured number with a source of
// the other three parts is read.
func TestAMeasuredNumberNeedsNoPage(t *testing.T) {
	file := without(sound, sourceLines["page"])
	read, err := numbers.Read("good.yaml", []byte(file))
	if err != nil {
		t.Fatal(err)
	}
	speed, err := read.Get("continuous-speed")
	if err != nil || speed.From == nil || speed.From.Key != "nbsir-79-1910" {
		t.Errorf("got %+v, %v", speed, err)
	}
}

// Read refuses a quoted name that is a number in another base. The reader
// strips the quotes before the code sees the name, so it cannot tell.
func TestAQuotedNameThatIsANumberInAnotherBaseIsRefused(t *testing.T) {
	for _, name := range []string{"'0x10'", "\"0X1f\"", "'0o7'", "'0b11'",
		"'-0x10'"} {
		_, err := readNamed("bad.yaml", name)
		if err == nil || !strings.Contains(err.Error(), "call the number "+
			"something else") {
			t.Errorf("%s was taken: %v", name, err)
		}
	}
}

// Anti-test: quoted names that are no number in any base are taken.
func TestAQuotedNameThatIsNoNumberIsTaken(t *testing.T) {
	for _, name := range []string{"'0xg'", "\"0o8\"", "'x0x10'"} {
		if _, err := readNamed("good.yaml", name); err != nil {
			t.Errorf("%s was refused: %v", name, err)
		}
	}
}
