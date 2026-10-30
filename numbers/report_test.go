package numbers_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"github.com/janearc/puzzlemouth/yamlfile"
)

// threeBad is a file of three entries, each with a different fault, and
// one entry that is sound.
const threeBad = `numbers:
  - name: guessed
    value: 1
    says: a kind that is neither
    kind: guessed
  - name: fine
    value: 2
    says: nothing wrong
    kind: picked
    by: j
  - name: unsourced
    value: 3
    says: measured with no source
    kind: measured
  - name: unpicked
    value: 4
    says: picked by no one
    kind: picked
`

// Number and Source are the types the file is read into. Read the file
// with the YAML reader straight into a list of them and the result is
// what numbers.Read returns, so nothing is copied from one type to
// another.
func TestNumberAndSourceAreFilledStraightFromTheFile(t *testing.T) {
	var file struct {
		Numbers numbers.Numbers `yaml:"numbers"`
	}
	if err := yamlfile.Read("sound.yaml", []byte(sound), &file); err != nil {
		t.Fatal(err)
	}
	want := numbers.Numbers{
		{Name: "continuous-speed", Value: 6.84, Unit: "m s-1 m-1/2",
			Says: "the centreline speed over the root of the height",
			Kind: numbers.Measured,
			From: &numbers.Source{Key: "nbsir-79-1910", Page: 35,
				Where: "table 1, pdf page 35, row v/z^1/2, weighted average",
				Text:  "6.84"}},
		{Name: "deepest", Value: 2,
			Says: "how many calls may stand inside one another",
			Kind: numbers.Picked, By: "ada, 2026-10-18"},
	}
	if !reflect.DeepEqual(file.Numbers, want) {
		t.Errorf("filled from the file:\n got %+v\nwant %+v", file.Numbers, want)
	}
	read, err := numbers.Read("sound.yaml", []byte(sound))
	if err != nil || !reflect.DeepEqual(read, want) {
		t.Errorf("numbers.Read:\n got %+v, %v\nwant %+v", read, err, want)
	}
}

// Anti-test: Read walks every entry. Three bad entries in one file are
// all reported, each named by its number, and the sound one is not.
func TestEveryBadEntryIsReportedTogether(t *testing.T) {
	read, err := numbers.Read("bad.yaml", []byte(threeBad))
	if err == nil {
		t.Fatal("a file of three bad entries was taken")
	}
	for _, name := range []string{"guessed", "unsourced", "unpicked"} {
		if !strings.Contains(err.Error(), name+":") {
			t.Errorf("the error does not name %s: %v", name, err)
		}
	}
	if strings.Contains(err.Error(), "fine") {
		t.Errorf("the error names the sound entry: %v", err)
	}
	if read != nil {
		t.Errorf("a refusal came back with numbers: %+v", read)
	}
}

// Anti-test: one of each kind of refusal Read makes, in one file: a bad
// kind, a measured number with no source, a picked one with no picker, a
// name written twice and a name the YAML reader would rewrite.
func TestEachKindOfRefusalIsReportedInOneFile(t *testing.T) {
	file := threeBad +
		"  - name: fine\n    value: 5\n    says: again\n    kind: picked\n    by: j\n" +
		"  - name: 010\n    value: 6\n    says: a number's name\n    kind: picked\n    by: j\n"
	read, err := numbers.Read("bad.yaml", []byte(file))
	if err == nil || read != nil {
		t.Fatalf("taken: %+v, %v", read, err)
	}
	for _, piece := range []string{"guessed:", "unsourced:", "unpicked:",
		"fine is written twice", "call the number something else"} {
		if !strings.Contains(err.Error(), piece) {
			t.Errorf("the error lacks %q: %v", piece, err)
		}
	}
}

// Anti-test: the file name is on the joined error once, not on each
// refusal.
func TestTheFileNameIsOnTheErrorOnce(t *testing.T) {
	_, err := numbers.Read("bad.yaml", []byte(threeBad))
	if err == nil {
		t.Fatal("a file of three bad entries was taken")
	}
	if got := strings.Count(err.Error(), "bad.yaml"); got != 1 {
		t.Errorf("the file is named %d times, not once: %v", got, err)
	}
}

// Anti-test: a refusal from the YAML reader is in the same error as
// Read's own, with its line. The file name is still there once.
func TestAReaderRefusalAndReadsOwnAreReportedTogether(t *testing.T) {
	file := strings.Replace(threeBad, "    says: nothing wrong\n",
		"    says: nothing wrong\n    colour: red\n", 1)
	read, err := numbers.Read("bad.yaml", []byte(file))
	if err == nil || read != nil {
		t.Fatalf("taken: %+v, %v", read, err)
	}
	if !strings.Contains(err.Error(), "line 9") ||
		!strings.Contains(err.Error(), "colour") {
		t.Errorf("the unknown key is not at its line: %v", err)
	}
	if got := strings.Count(err.Error(), "bad.yaml"); got != 1 {
		t.Errorf("the file is named %d times, not once: %v", got, err)
	}
}

// Anti-test: an empty file holds no numbers and is not a refusal. A file
// with a numbers key and no entries is the same.
func TestAFileWithNoNumbersIsNoRefusal(t *testing.T) {
	for _, empty := range []string{"", "numbers: []\n", "# nothing\n"} {
		read, err := numbers.Read("empty.yaml", []byte(empty))
		if err != nil || len(read) != 0 {
			t.Errorf("%q: %+v, %v", empty, read, err)
		}
	}
}
