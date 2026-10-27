package config_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
	"pgregory.net/rapid"
)

// noKey are lines that say something after the divider and have no
// setting name before it. Each is spaced with something other than the
// space and tab that indentation is made of.
var noKey = []string{
	"  :: payload",      // no-break space
	"\v :: payload",     // vertical tab
	"\f :: payload",     // form feed
	"\u0085 :: payload", // next line
	"  :: payload",      // em space
	"   :: x :: y",      // two, and a tail that holds a divider
}

const (
	// fileName is the name of the file a test reads. It's defined.
	fileName = "settings.conf"
	// badLine is the line a bad line is put on, counted from one. It's
	// defined.
	badLine = "2"
	// afterLine is the line after it, which is bad too. It's defined.
	afterLine = "3"
)

// Read bounces a divider with no setting before it, by file and line.
// It does not panic, and the file is bounced whole.
func TestALineWithNoKeyIsBounced(t *testing.T) {
	for _, bad := range noKey {
		for _, file := range []string{
			"model one\n" + bad + "\nmodel two three\n",
			"project pzm\n  " + bad + "\nmodel two three\n"} {
			read, err := config.Read(fileName, []byte(file), settings(t)...)
			if read != nil || err == nil {
				t.Fatalf("%q: read %v, err %v", file, read, err)
			}
			for _, line := range []string{badLine, afterLine} {
				if !strings.Contains(err.Error(), fileName+":"+line+": ") {
					t.Errorf("%q: line %s is not named in:\n%v", file,
						line, err)
				}
			}
		}
	}
}

// Anti-test: a line with a key and a divider is read as before, and one
// with only the divider after a key that takes a tail is still taken.
func TestALineWithAKeyAndATailIsStillRead(t *testing.T) {
	file := "lint ninja :: perl -e x\r\nproject pzm\r\n  dist none\r\n"
	read, err := config.Read(fileName, []byte(file), settings(t)...)
	if err != nil || !slices.Equal(tails(read), []string{"perl -e x", ""}) {
		t.Errorf("read %v, err %v", read, err)
	}
}

// tails are the tails of the entries, in order.
func tails(read []config.Entry) []string {
	var all []string
	for _, each := range read {
		all = append(all, each.Tail())
	}
	return all
}

// pieces build the files of the property below. Many are spaces of odd
// kinds. One is the divider. The rest are setting names and line ends.
var pieces = []string{" ", "\t", " ", "\v", "\f", "\u0085", " ",
	" :: ", "::", "=", "#", "model", "lint", "project", "dist", "x",
	"\n", "\n  ", "\r\n"}

// bounceLine is how each line of a bounce begins.
var bounceLine = regexp.MustCompile(`^` + fileName + `:[0-9]+: `)

// Reading any text either gives entries and no error, or gives no
// entries and an error whose every line names the file and a line of it.
// It never panics.
func TestAnyTextIsReadOrBounced(t *testing.T) {
	rapid.Check(t, checkReadOrBounce)
}

// checkReadOrBounce is one draw of that property.
func checkReadOrBounce(t *rapid.T) {
	text := strings.Join(rapid.SliceOf(rapid.SampledFrom(pieces)).Draw(t,
		"pieces"), "")
	read, err := config.Read(fileName, []byte(text), settings(t)...)
	switch {
	case err == nil:
		return
	case read != nil:
		t.Fatalf("%q: bounced, and still read %v", text, read)
	}
	for _, line := range strings.Split(err.Error(), "\n") {
		if !bounceLine.MatchString(line) {
			t.Fatalf("%q: a bounce line that names no line: %q", text,
				line)
		}
	}
}

// Anti-test over bytes that are not text: Read never bounces and reads.
func TestAnyBytesAreReadOrBounced(t *testing.T) {
	rapid.Check(t, checkBytesReadOrBounce)
}

// checkBytesReadOrBounce is one draw of that property.
func checkBytesReadOrBounce(t *rapid.T) {
	raw := rapid.SliceOf(rapid.Byte()).Draw(t, "bytes")
	read, err := config.Read(fileName, raw, settings(t)...)
	if err != nil && read != nil {
		t.Fatalf("%q: bounced, and still read %v", raw, read)
	}
}
