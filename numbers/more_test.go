package numbers_test

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
	"pgregory.net/rapid"
)

// Every part of a number is read back: where and what it says as well as
// the rest.
func TestEveryPartOfANumberIsReadBack(t *testing.T) {
	read, err := numbers.Read("sound.yaml", []byte(sound))
	if err != nil {
		t.Fatal(err)
	}
	speed, err := read.Get("continuous-speed")
	if err != nil || speed.From == nil {
		t.Fatalf("the measured number: %+v, %v", speed, err)
	}
	if speed.Name != "continuous-speed" || speed.Says != "the centreline"+
		" speed over the root of the height" {
		t.Errorf("name %q, says %q", speed.Name, speed.Says)
	}
	if where := speed.From.Where; where != "table 1, pdf page 35, row "+
		"v/z^1/2, weighted average" {
		t.Errorf("where %q", where)
	}
}

// Anti-test: a number picked has no source, and one measured has no
// picker.
func TestAKindHasOnlyItsOwnParts(t *testing.T) {
	read, err := numbers.Read("sound.yaml", []byte(sound))
	if err != nil {
		t.Fatal(err)
	}
	speed, speedErr := read.Get("continuous-speed")
	deepest, deepestErr := read.Get("deepest")
	if speedErr != nil || deepestErr != nil {
		t.Fatalf("both are in the file: %v, %v", speedErr, deepestErr)
	}
	if speed.By != "" || deepest.From != nil {
		t.Errorf("measured by %q; picked from %+v", speed.By, deepest.From)
	}
}

// Anti-test: a file refuses each part a number is missing or has too
// many of, and names the number.
func TestEachMissingOrExtraPartIsRefused(t *testing.T) {
	for why, bad := range map[string]struct{ file, says string }{
		"no name": {"numbers:\n  - value: 1\n    says: x\n    kind: " +
			"picked\n    by: j\n", "no name"},
		"no says": {"numbers:\n  - name: a\n    value: 1\n    kind: " +
			"picked\n    by: j\n", "a: a number needs"},
		"measured with a picker": {strings.ReplaceAll(sound,
			"      text: \"6.84\"\n", "      text: \"6.84\"\n    by: j\n"),
			"continuous-speed: a measured number names"},
		"measured with no text": {strings.ReplaceAll(sound,
			"      text: \"6.84\"\n", ""),
			"continuous-speed: a measured number names"},
		"picked with a source": {strings.ReplaceAll(sound,
			"    by: ada, 2026-10-18\n", "    by: j\n    source:\n"+
				"      key: k\n"), "deepest: a picked number says"},
	} {
		read, err := numbers.Read("bad.yaml", []byte(bad.file))
		if err == nil || !strings.Contains(err.Error(), bad.says) ||
			!strings.Contains(err.Error(), "bad.yaml") {
			t.Errorf("%s: %v", why, err)
		}
		if read != nil {
			t.Errorf("%s came back with numbers: %+v", why, read)
		}
	}
}

// Anti-test: text that is not YAML at all is refused, naming the file.
func TestTextThatIsNotYAMLIsRefused(t *testing.T) {
	for _, bad := range []string{"numbers: [", "a: b: c", "\t- x",
		"numbers: 3", "numbers:\n  - 3\n"} {
		read, err := numbers.Read("bad.yaml", []byte(bad))
		if err == nil || !strings.Contains(err.Error(), "bad.yaml") ||
			read != nil {
			t.Errorf("%q: %v, %v", bad, read, err)
		}
	}
}

const (
	// unit is the value of the numbers a test only names. It's defined.
	unit = 1
	// shortest asks FormatFloat for the fewest digits that read back as
	// the same number. It's defined by strconv.FormatFloat: -1.
	shortest = -1
	// wide is the size of a float64 in bits. It's defined.
	wide = 64
)

// namedFile is a file with one picked number called name, written as it
// is given.
func namedFile(name string) string {
	return pickedFile([]string{name}, []float64{unit})
}

// readNamed is namedFile read.
func readNamed(file, name string) (numbers.Numbers, error) {
	text := []byte(namedFile(name))
	return numbers.Read(file, text)
}

// readDrawn is the picked numbers read, as a file called drawn.yaml.
func readDrawn(names []string, values []float64) (numbers.Numbers, error) {
	text := []byte(pickedFile(names, values))
	return numbers.Read("drawn.yaml", text)
}

// namesOf is the names of the numbers, in order.
func namesOf(read numbers.Numbers) []string {
	var names []string
	for _, each := range read {
		names = append(names, each.Name)
	}
	return names
}

// Anti-test: a name that parses as a number in any base is refused, for
// the reader of the code: nobody should have to ask for Get("010").
func TestANameThatParsesAsANumberIsRefused(t *testing.T) {
	for _, name := range []string{"010", "0x10", "1e3", ".5", "1_000",
		"7"} {
		_, err := readNamed("bad.yaml", name)
		if err == nil || !strings.Contains(err.Error(), "call the number "+
			"something else") {
			t.Errorf("%s was taken: %v", name, err)
		}
	}
}

// Names the reader keeps are taken as written. yaml.v3 keeps yes, on, no,
// off, true and false as the strings they were written as, so they are
// no longer refused. Quotes make a word of YAML text, so those names are
// taken too.
func TestANameTheReaderKeepsIsTaken(t *testing.T) {
	for written, kept := range map[string]string{"deepest": "deepest",
		"on-air": "on-air", "3d-scale": "3d-scale", "inf": "inf",
		"nan": "nan", "e2": "e2", "'yes'": "yes", "\"no\"": "no",
		"y": "y", "yes": "yes", "on": "on", "n": "n", "no": "no",
		"off": "off", "true": "true", "false": "false"} {
		read, err := readNamed("good.yaml", written)
		if err != nil || !slices.Equal(namesOf(read), []string{kept}) {
			t.Errorf("%s: %v, %v", written, namesOf(read), err)
		}
	}
}

const (
	// fewest is the fewest numbers a drawn file holds. It's defined.
	fewest = 1
	// most is the most. It's defined.
	most = 6
	// highest bounds the values drawn, both ways. It's defined.
	highest = 1e6
)

// pickedFile is the YAML of picked numbers. Each name gets one.
func pickedFile(names []string, values []float64) string {
	var file strings.Builder
	file.WriteString("numbers:\n")
	for place, name := range names {
		fmt.Fprintf(&file, "  - name: %s\n    value: %s\n    says: "+
			"what %s is\n    kind: picked\n    by: j\n", name,
			strconv.FormatFloat(values[place], 'f', shortest, wide), name)
	}
	return file.String()
}

// words are names the YAML reader makes of other words, and the plain
// words of YAML that are no name.
var words = []string{"y", "n", "yes", "no", "on", "off", "true", "false",
	"null"}

// notAWord is whether a drawn name is none of the words.
func notAWord(name string) bool { return !slices.Contains(words, name) }

// genNames draws distinct names that the reader keeps as written.
func genNames(t *rapid.T) []string {
	name := rapid.StringMatching(`[a-z][a-z0-9]{0,5}(-[a-z0-9]{1,4})?`).
		Filter(notAWord)
	return rapid.SliceOfNDistinct(name, fewest, most,
		rapid.ID[string]).Draw(t, "names")
}

// genValues draws a value for each name.
func genValues(t *rapid.T, count int) []float64 {
	value := rapid.Float64Range(-highest, highest)
	return rapid.SliceOfN(value, count, count).Draw(t, "values")
}

// Picked numbers are read back by name with their values. All has them
// in the order written.
func TestPickedNumbersAreReadBack(t *testing.T) {
	rapid.Check(t, checkReadBack)
}

// checkReadBack is one draw of that property.
func checkReadBack(t *rapid.T) {
	names := genNames(t)
	values := genValues(t, len(names))
	read, err := readDrawn(names, values)
	if err != nil {
		t.Fatal(err)
	}
	for place, name := range names {
		got, err := read.Get(name)
		if err != nil || got.Value != values[place] ||
			got.Kind != numbers.Picked {
			t.Fatalf("%s: %+v, %v", name, got, err)
		}
	}
	if order := namesOf(read); !slices.Equal(order, names) {
		t.Fatalf("order %q, written %q", order, names)
	}
}

// Anti-test: a name written twice is refused wherever the two are, and
// no number is read from the file.
func TestANameWrittenTwiceIsRefusedAnywhere(t *testing.T) {
	rapid.Check(t, checkTwice)
}

// checkTwice is one draw of that property.
func checkTwice(t *rapid.T) {
	names := genNames(t)
	copied := rapid.SampledFrom(names).Draw(t, "copied")
	names = append(names, copied)
	values := genValues(t, len(names))
	read, err := readDrawn(names, values)
	if err == nil || !strings.Contains(err.Error(), "written twice") ||
		read != nil {
		t.Fatalf("%q: %v", names, err)
	}
}
