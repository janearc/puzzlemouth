package yamlfile_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/yamlfile"
)

// member is one element of a list in the files these tests read.
type member struct {
	Name  string `yaml:"name"`
	Count int    `yaml:"count"`
}

// shape is the type the files in these tests are read into.
type shape struct {
	Title   string   `yaml:"title"`
	Ratio   float64  `yaml:"ratio"`
	Quiet   bool     `yaml:"quiet"`
	Members []member `yaml:"members"`
	Inner   *member  `yaml:"inner"`
}

// good is a file that fills shape.
const good = `# a comment
title: the lodge
ratio: 6.84
quiet: true
members:
  - name: ada
    count: 2

  - name: grace
    count: 3
inner:
  name: edsger
  count: 4
`

// readInto is raw read into a fresh shape, as the file bad.yaml.
func readInto(raw string) (shape, error) {
	var into shape
	err := yamlfile.Read("bad.yaml", []byte(raw), &into)
	return into, err
}

// wants fails the test unless err names the file once and holds every one
// of the pieces.
func wants(t *testing.T, err error, pieces ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the file was taken, wanted %q", pieces)
	}
	if got := strings.Count(err.Error(), "bad.yaml"); got != 1 {
		t.Errorf("the file is named %d times, not once: %v", got, err)
	}
	for _, piece := range pieces {
		if !strings.Contains(err.Error(), piece) {
			t.Errorf("the error lacks %q: %v", piece, err)
		}
	}
}

func TestAGoodFileFillsTheType(t *testing.T) {
	var into shape
	if err := yamlfile.Read("good.yaml", []byte(good), &into); err != nil {
		t.Fatal(err)
	}
	if into.Title != "the lodge" || into.Ratio != 6.84 || !into.Quiet {
		t.Errorf("the plain fields: %+v", into)
	}
	if len(into.Members) != 2 || into.Members[0] != (member{"ada", 2}) ||
		into.Members[1] != (member{"grace", 3}) {
		t.Errorf("the list, in the order written: %+v", into.Members)
	}
	if into.Inner == nil || *into.Inner != (member{"edsger", 4}) {
		t.Errorf("the nested value: %+v", into.Inner)
	}
}

// A key the file leaves out keeps what the caller put there.
func TestAKeyTheFileLeavesOutKeepsTheCallersDefault(t *testing.T) {
	into := shape{Title: "unnamed", Ratio: 1}
	if err := yamlfile.Read("good.yaml", []byte("ratio: 2\n"), &into); err != nil {
		t.Fatal(err)
	}
	if into.Title != "unnamed" || into.Ratio != 2 {
		t.Errorf("default kept, key set: %+v", into)
	}
}

// Anti-test: a key the type has no field for is refused at its line.
func TestAnUnknownKeyIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto("title: x\n\n# note\ncolour: red\n")
	wants(t, err, "line 4", "colour")
}

// Anti-test: an unknown key inside a list is found at its own line.
func TestAnUnknownKeyInAListIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto("members:\n  - name: a\n    count: 1\n\n  - name: b\n    nick: c\n")
	wants(t, err, "line 6", "nick")
}

// Anti-test: a key written twice is refused at the second line, and the
// error says which key.
func TestAKeyWrittenTwiceIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto("title: a\nratio: 1\ntitle: b\n")
	wants(t, err, "line 3", "title", "already")
}

// Anti-test: a value that cannot be its field's type is refused at its
// line.
func TestAValueOfTheWrongTypeIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto("title: x\nratio: fast\n")
	wants(t, err, "line 2", "fast")
}

// Anti-test: a value of the wrong type inside a list is found at its own
// line, past comments and blank lines.
func TestAWrongTypeInAListIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto("# one\n\nmembers:\n  - name: a\n    count: many\n")
	wants(t, err, "line 5", "many")
}

// Anti-test: unknown keys and wrong types are all reported, each at its
// line, with the file named once.
func TestEveryRefusalIsReportedTogether(t *testing.T) {
	_, err := readInto("title: x\nratio: fast\ncolour: red\n" +
		"members:\n  - name: a\n    count: many\n    nick: b\n")
	wants(t, err, "line 2", "line 3", "line 6", "line 7")
}

// Anti-test: every kind of refusal is an error, and the destination is
// left as the caller had it, not partly filled: the good lines before the
// bad one are not kept.
func TestEveryKindOfRefusalIsAnErrorAndLeavesIntoUntouched(t *testing.T) {
	for why, bad := range map[string]string{
		"an unknown key":    "title: a\ncolour: red\n",
		"a duplicate key":   "title: a\ntitle: b\n",
		"a wrong type":      "title: a\nratio: fast\n",
		"not YAML":          "title: a\nquiet: [\n",
		"a second document": "title: a\n---\ntitle: b\n",
	} {
		into := shape{Title: "unnamed", Members: []member{{"keep", 1}}}
		err := yamlfile.Read("bad.yaml", []byte(bad), &into)
		if err == nil {
			t.Errorf("%s was taken", why)
		}
		if into.Title != "unnamed" || len(into.Members) != 1 ||
			into.Members[0] != (member{"keep", 1}) {
			t.Errorf("%s changed the destination: %+v", why, into)
		}
	}
}

// An empty file, or one of only comments, is no refusal and fills nothing.
func TestAnEmptyFileFillsNothing(t *testing.T) {
	for _, empty := range []string{"", "\n\n", "# only a comment\n"} {
		into := shape{Title: "unnamed"}
		if err := yamlfile.Read("empty.yaml", []byte(empty), &into); err != nil {
			t.Errorf("%q was refused: %v", empty, err)
		}
		if into.Title != "unnamed" {
			t.Errorf("%q changed the value: %+v", empty, into)
		}
	}
}

// Anti-test: text that is not YAML is refused, naming the file. The YAML
// library gives a line for some of these faults and not for others, so
// only the first is held to a line.
func TestTextThatIsNotYAMLIsRefused(t *testing.T) {
	_, err := readInto("title: [")
	wants(t, err, "line 1")
	for _, bad := range []string{"a: b: c", "\t- x", "title: 'x"} {
		_, err := readInto(bad)
		wants(t, err)
	}
}

// Anti-test: text that is YAML but not a mapping cannot fill a struct.
func TestYAMLThatIsNotAMappingIsRefused(t *testing.T) {
	for _, bad := range []string{"- a\n- b\n", "just text\n", "3\n"} {
		_, err := readInto(bad)
		wants(t, err)
	}
}

// Anti-test: a nil destination is refused, not a panic.
func TestANilDestinationIsRefused(t *testing.T) {
	if err := yamlfile.Read[shape]("bad.yaml", []byte(good), nil); err == nil {
		t.Error("a nil destination was taken")
	}
}

// Words the YAML 1.1 readers turn into other values are kept as written
// when the field is a string. The rule that numbers.Read refuses such
// names belongs to numbers: it is about which names are confusing, and
// this reader does not rewrite anything.
func TestAStringFieldKeepsWhatWasWritten(t *testing.T) {
	for _, word := range []string{"yes", "on", "no", "true", "false", "010",
		"0x10", "0o7", "1e3", ".5", "1_000"} {
		var into shape
		err := yamlfile.Read("word.yaml", []byte("title: "+word+"\n"), &into)
		if err != nil || into.Title != word {
			t.Errorf("%s was read as %q: %v", word, into.Title, err)
		}
	}
}
