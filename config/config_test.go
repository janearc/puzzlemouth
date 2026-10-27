package config_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// failer is what settings needs of a test: both testing.T and rapid.T
// are one.
type failer interface {
	Helper()
	Fatal(args ...any)
}

// settings are a test tool's: a model name, an address, a lint line with
// a command as its tail, and a project block with a road and a dist.
func settings(t failer) []config.Setting {
	t.Helper()
	var declared []config.Setting
	for _, made := range []func() (config.Setting, error){
		func() (config.Setting, error) {
			return config.New("model", "the model to run", oneWord, config.NoTail)
		},
		func() (config.Setting, error) {
			return config.New("listen", "where to listen", oneWord, config.NoTail)
		},
		func() (config.Setting, error) {
			return config.New("lint", "a linter, by name, and its command", oneWord,
				config.Tail)
		},
		func() (config.Setting, error) {
			road, err := config.New("road", "the dirty remote", oneWord, config.NoTail)
			if err != nil {
				return config.Setting{}, err
			}
			dist, err := config.New("dist", "the clean remote", oneWord, config.NoTail)
			if err != nil {
				return config.Setting{}, err
			}
			return config.Block("project", "a project, by name", oneWord, road, dist)
		},
	} {
		setting, err := made()
		if err != nil {
			t.Fatal(err)
		}
		declared = append(declared, setting)
	}
	return declared
}

// sound is a file that conforms: every line declared, and every kind of
// line there is.
const sound = `# a comment
model tiny-14b-12k
listen = ::1

lint ninja :: perl -MFoo::Bar -e 'run()' :: whatever
project pzm
  road ~/roots/puzzlemouth-road.git
  # a comment inside a block
  dist none
`

// A file that conforms is read line by line: each key with its words,
// its tail, and the lines of its block.
func TestASoundFileIsRead(t *testing.T) {
	read, err := config.Read("config", []byte(sound), settings(t)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != 4 {
		t.Fatalf("read %d entries, want 4", len(read))
	}
	said := func(each config.Entry) string {
		return each.Key() + " " + strings.Join(each.Words(), ",") + " | " +
			each.Tail()
	}
	for at, want := range []string{"model tiny-14b-12k | ", "listen ::1 | ",
		"lint ninja | perl -MFoo::Bar -e 'run()' :: whatever", "project pzm | "} {
		if got := said(read[at]); got != want {
			t.Errorf("entry %d is %q, want %q", at, got, want)
		}
	}
	inside := read[3].In()
	if len(inside) != 2 || said(inside[0]) != "road ~/roots/puzzlemouth-road.git | " ||
		said(inside[1]) != "dist none | " {
		t.Errorf("the project block held %v", inside)
	}
	if read[2].Line() != 5 || inside[1].Line() != 9 {
		t.Errorf("lines %d and %d, want 5 and 9", read[2].Line(), inside[1].Line())
	}
}

// A # is a comment only as the first thing on a line, so a value may
// hold one.
func TestAHashInAValueIsKept(t *testing.T) {
	read, err := config.Read("config", []byte("listen http://x/#here\n"),
		settings(t)...)
	if err != nil || len(read) != single || read[0].Words()[0] != "http://x/#here" {
		t.Errorf("read %v, %v", read, err)
	}
}

// A file that does not conform is bounced whole: nothing is read from it,
// and every line that is wrong is named with its file and line.
func TestAFileThatDoesNotConformIsBounced(t *testing.T) {
	bad := strings.Join([]string{
		"modle tiny",        // 1: not a setting
		"model one two",     // 2: two words where one is declared
		"model one :: tail", // 3: a tail where none is declared
		"lint ninja",        // 4: no tail where one is declared
		"  road ~/x",        // 5: inside no block
		"project pzm",       // 6: a sound block
		"\troad ~/x",        // 7: indented with a tab
		"  model one",       // 8: not a setting of a project
		"lint :: perl -e x", // 9: no name before the tail
		"model",             // 10: no words at all
	}, "\n")
	read, err := config.Read("settings.conf", []byte(bad), settings(t)...)
	if read != nil {
		t.Errorf("a bounced file gave %d entries", len(read))
	}
	if err == nil {
		t.Fatal("the file was not bounced")
	}
	for _, line := range []string{"1", "2", "3", "4", "5", "7", "8", "9", "10"} {
		if !strings.Contains(err.Error(), "settings.conf:"+line+":") {
			t.Errorf("line %s is not named in:\n%v", line, err)
		}
	}
	if strings.Contains(err.Error(), "settings.conf:6:") {
		t.Errorf("the sound line 6 was named in:\n%v", err)
	}
	if !strings.Contains(err.Error(), "model, listen, lint, project") {
		t.Errorf("the bounce does not list the settings there are:\n%v", err)
	}
}

// A setting is declared with a name, a line of help, how many words it
// takes and whether it takes a tail; one that cannot be is refused.
func TestASettingIsDeclaredWhole(t *testing.T) {
	for why, made := range map[string]func() (config.Setting, error){
		"a name with capitals": func() (config.Setting, error) {
			return config.New("Model", "h", oneWord, config.NoTail)
		},
		"no help": func() (config.Setting, error) {
			return config.New("model", "", oneWord, config.NoTail)
		},
		"no words and no tail": func() (config.Setting, error) {
			return config.New("model", "h", noWords, config.NoTail)
		},
		"fewer than no words": func() (config.Setting, error) {
			return config.New("model", "h", -1, config.Tail)
		},
		"a block holding nothing": func() (config.Setting, error) {
			return config.Block("project", "h", 1)
		},
	} {
		if _, err := made(); err == nil {
			t.Errorf("%s was declared", why)
		}
	}
}

// A bounce says what is wrong in words: how many words a setting takes,
// and, inside a block, the settings the block holds.
func TestABounceSaysWhatIsWrong(t *testing.T) {
	pair, err := config.New("pair", "two words", twoWords, config.NoTail)
	if err != nil {
		t.Fatal(err)
	}
	declared := append(settings(t), pair)
	for text, want := range map[string]string{
		"model one two":            "config:1: model takes 1 word, and this line has 2",
		"pair one":                 "config:1: pair takes 2 words, and this line has 1",
		"model =":                  "config:1: model takes 1 word, and this line has 0",
		"project pzm\n  model one": `config:2: "model" is not a setting of project; its settings are road, dist`,
		"  road ~/x":               "config:1: it is indented, but under no block",
		"modle tiny":               `config:1: "modle" is not a setting; the settings are model,`,
	} {
		_, err := config.Read("config", []byte(text), declared...)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: bounced with %v, want %q", text, err, want)
		}
	}
}

// A setting may take a tail and no words, for a line that is all reason.
func TestASettingMayBeAllTail(t *testing.T) {
	note, err := config.New("note", "a note to keep", noWords, config.Tail)
	if err != nil {
		t.Fatal(err)
	}
	read, err := config.Read("config", []byte("note :: anything at all"), note)
	if err != nil || len(read) != single || read[0].Tail() != "anything at all" {
		t.Errorf("read %v, %v", read, err)
	}
}

// A tail is trimmed at both ends, and an empty one is said as a missing
// tail, not miscounted as a word..
func TestATailIsTrimmedAndAnEmptyOneIsSaid(t *testing.T) {
	note, err := config.New("note", "a note to keep", noWords, config.Tail)
	if err != nil {
		t.Fatal(err)
	}
	run, err := config.New("run", "a target and its command", oneWord, config.Tail)
	if err != nil {
		t.Fatal(err)
	}
	read, err := config.Read("config", []byte("run x :: y  \r"), note, run)
	if err != nil || len(read) != single || read[0].Tail() != "y" {
		t.Errorf("trailing spaces: read %v, %v", read, err)
	}
	for _, line := range []string{"run x :: ", "note ::", "note :: \r"} {
		_, err := config.Read("config", []byte(line), note, run)
		if err == nil || !strings.Contains(err.Error(), "takes a tail") {
			t.Errorf("%q: %v, want it bounced as taking a tail", line, err)
		}
	}
	listen, err := config.New("listen", "where to listen", oneWord, config.NoTail)
	if err != nil {
		t.Fatal(err)
	}
	read, err = config.Read("config", []byte("listen ::"), listen)
	if err != nil || len(read) != single || read[0].Words()[0] != "::" {
		t.Errorf("listen ::, any address, read %v, %v", read, err)
	}
}
