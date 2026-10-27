package config_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
	"pgregory.net/rapid"
)

const (
	// twoWords is how many words the second test setting takes. It's
	// defined.
	twoWords = 2
	// threeWords is how many words the third takes. It's defined.
	threeWords = 3
	// mostWords is the most words a drawn line has. It's defined.
	mostWords = 5
	// noWords is a line of just its key. It's defined.
	noWords = 0
)

// takes is a setting of the test that takes a number of words, and how
// many that is.
type takes struct {
	key     string
	setting config.Setting
	words   int
}

// byWords are one setting for each number of words, with the number.
func byWords() []takes {
	one := must(config.New("one", "takes one", oneWord, config.NoTail))
	two := must(config.New("two", "takes two", twoWords, config.NoTail))
	three := must(config.New("three", "takes three", threeWords,
		config.NoTail))
	return []takes{{"one", one, oneWord}, {"two", two, twoWords},
		{"three", three, threeWords}}
}

// Read takes a line exactly when it has the words its setting takes. Any
// other number is bounced. The bounce says how many it takes and how
// many it was given.
func TestALineHasExactlyTheWordsItsSettingTakes(t *testing.T) {
	rapid.Check(t, checkWords)
}

// checkWords is one draw of that property.
func checkWords(t *rapid.T) {
	chosen := rapid.SampledFrom(byWords()).Draw(t, "setting")
	given := rapid.IntRange(noWords, mostWords).Draw(t, "given")
	line := chosen.key + strings.Repeat(" w", given)
	read, err := config.Read(fileName, []byte(line+"\n"), chosen.setting)
	if given == chosen.words {
		if err != nil || len(read) != oneWord ||
			len(read[firstEntry].Words()) != given {
			t.Fatalf("%q: read %v, err %v", line, read, err)
		}
		return
	}
	want := "takes " + counted(chosen.words) + ", and this line has " +
		strconv.Itoa(given)
	if read != nil || err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%q: read %v, err %v; want %q", line, read, err, want)
	}
}

// firstEntry is the first entry of a file. It's defined.
const firstEntry = 0

// counted is a number of words as the bounce says it.
func counted(words int) string {
	if words == oneWord {
		return "1 word"
	}
	return strconv.Itoa(words) + " words"
}

// indents are the ways a line in a block may be indented, and whether
// each is allowed: spaces are and a tab anywhere is not.
var indents = map[string]bool{"  ": true, " ": true, "    ": true,
	"\t": false, " \t": false, "\t ": false, "  \t": false}

// Anti-test: a tab in the indent bounces the line, with spaces or none.
// Spaces alone are read.
func TestATabInTheIndentIsBounced(t *testing.T) {
	rapid.Check(t, checkIndent)
}

// checkIndent is one draw of that property.
func checkIndent(t *rapid.T) {
	chosen := rapid.SampledFrom([]string{"  ", " ", "    ", "\t", " \t",
		"\t ", "  \t"}).Draw(t, "indent")
	file := "alpha a\n" + chosen + "child 1\n"
	read, err := config.Read(fileName, []byte(file), twoBlocks()...)
	if indents[chosen] {
		if err != nil || len(read) != oneWord {
			t.Fatalf("%q: read %v, err %v", chosen, read, err)
		}
		return
	}
	if read != nil || err == nil || !strings.Contains(err.Error(),
		"indented with a tab") {
		t.Fatalf("%q: read %v, err %v", chosen, read, err)
	}
}
