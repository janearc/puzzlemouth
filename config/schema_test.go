package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
	"pgregory.net/rapid"
)

// oneWord is how many words the test settings take. It's defined.
const oneWord = 1

// must is made, or a panic that fails the test.
func must(made config.Setting, err error) config.Setting {
	if err != nil {
		panic(err)
	}
	return made
}

// leaf is a one-word setting of a test, called key. Any key it does not
// know is child.
func leaf(key string) config.Setting {
	switch key {
	case "model":
		return must(config.New("model", "a model", oneWord,
			config.NoTail))
	case "listen":
		return must(config.New("listen", "an address", oneWord,
			config.NoTail))
	case "lint":
		return must(config.New("lint", "a linter", oneWord, config.Tail))
	case "other":
		return must(config.New("other", "another", oneWord,
			config.NoTail))
	}
	return must(config.New("child", "a child", oneWord, config.NoTail))
}

// Block refuses a block among its settings and names both. The reader
// keeps one block open, so the inner block could be declared and never
// read.
func TestABlockCannotHoldABlock(t *testing.T) {
	inner := leaf("model")
	child := must(config.Block("child", "a child", oneWord, inner))
	_, err := config.Block("parent", "a parent", oneWord, child)
	if err == nil || !strings.Contains(err.Error(), "parent") ||
		!strings.Contains(err.Error(), "child") {
		t.Errorf("a block holding a block was made: %v", err)
	}
}

// Anti-test: a block that holds settings only is made and read.
func TestABlockOfSettingsIsMadeAndRead(t *testing.T) {
	child, other := leaf("child"), leaf("other")
	block := must(config.Block("parent", "a parent", oneWord, child,
		other))
	file := []byte("parent x\n  child y\n  other z\n")
	read, err := config.Read(fileName, file, block)
	if err != nil || !slices.Equal(flat(read),
		[]string{"parent", "child", "other"}) {
		t.Errorf("read %v, err %v", read, err)
	}
}

// flat is the keys of the entries, each followed by the keys under it.
func flat(read []config.Entry) []string {
	var keys []string
	for _, each := range read {
		under := flat(each.In())
		keys = append(keys, each.Key())
		keys = append(keys, under...)
	}
	return keys
}

// Block refuses a key twice. The first would be read, the second never.
func TestABlockCannotHoldAKeyTwice(t *testing.T) {
	_, err := config.Block("parent", "a parent", oneWord,
		leaf("child"), leaf("child"))
	if err == nil || !strings.Contains(err.Error(), "child") {
		t.Errorf("a block with two child settings was made: %v", err)
	}
}

// Block refuses a setting nobody made, which has no key.
func TestABlockCannotHoldAnUnmadeSetting(t *testing.T) {
	_, err := config.Block("parent", "a parent", oneWord, config.Setting{})
	if err == nil {
		t.Error("a block holding a setting nobody made was made")
	}
}

// Read refuses settings declared twice or not made. It reads nothing,
// even from an empty file.
func TestReadRefusesSettingsDeclaredWrong(t *testing.T) {
	for why, declared := range map[string][]config.Setting{
		"twice":  {leaf("model"), leaf("model")},
		"unmade": {leaf("model"), {}},
	} {
		read, err := config.Read(fileName, []byte(""), declared...)
		if read != nil || err == nil || !strings.Contains(err.Error(),
			fileName) {
			t.Errorf("%s: read %v, err %v", why, read, err)
		}
	}
}

// The caller owns the settings given to Block. Clearing them afterwards
// changes nothing the block reads.
func TestABlockKeepsItsOwnSettings(t *testing.T) {
	inner := []config.Setting{leaf("child")}
	block := must(config.Block("parent", "a parent", oneWord, inner...))
	clear(inner)
	read, err := config.Read(fileName, []byte("parent x\n  child y\n"), block)
	if err != nil || !slices.Equal(flat(read), []string{"parent", "child"}) {
		t.Errorf("read %v, err %v", read, err)
	}
}

// keyPool are the keys the property below declares settings with. Few,
// so that a key is often declared twice.
var keyPool = []string{"model", "listen", "lint"}

const (
	// fewest is the fewest settings the property declares. It's defined.
	fewest = 1
	// most is the most settings the property declares. It's defined.
	most = 5
)

// Read refuses settings when two share a key, and only then.
func TestSettingsAreRefusedExactlyWhenAKeyIsTwice(t *testing.T) {
	rapid.Check(t, checkKeysTwice)
}

// checkKeysTwice is one draw of that property.
func checkKeysTwice(t *rapid.T) {
	keys := rapid.SliceOfN(rapid.SampledFrom(keyPool), fewest,
		most).Draw(t, "keys")
	var declared []config.Setting
	seen := map[string]bool{}
	twice := false
	for _, key := range keys {
		declared = append(declared, leaf(key))
		twice = twice || seen[key]
		seen[key] = true
	}
	_, err := config.Read(fileName, []byte(""), declared...)
	if (err != nil) != twice {
		t.Fatalf("%v: twice %v, err %v", keys, twice, err)
	}
}
