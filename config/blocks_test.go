package config_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// twoBlocks are two blocks that hold different settings, and a plain
// setting between where they may be.
func twoBlocks() []config.Setting {
	child, other := leaf("child"), leaf("other")
	alpha := must(config.Block("alpha", "the first", oneWord, child))
	beta := must(config.Block("beta", "the second", oneWord, other))
	return []config.Setting{alpha, beta, leaf("model")}
}

// Indented lines belong to the block above them, and each block holds
// its own. Entries come in file order, a block's lines under its entry.
func TestEachBlockHoldsItsOwnLines(t *testing.T) {
	file := "alpha a\n  child 1\nbeta b\n  other 2\n  other 3\nmodel m\n"
	read, err := config.Read(fileName, []byte(file), twoBlocks()...)
	if err != nil || !slices.Equal(flat(read),
		[]string{"alpha", "child", "beta", "other", "other", "model"}) {
		t.Fatalf("read %v, err %v", read, err)
	}
}

// The left margin closes the block above it. After a plain setting an
// indented line is under no block. After the second block opens it is
// read against the second block.
func TestAnUnindentedLineClosesTheBlock(t *testing.T) {
	file := "alpha a\n  child 1\nmodel m\n  child 2\nbeta b\n  child 3\n"
	read, err := config.Read(fileName, []byte(file), twoBlocks()...)
	if read != nil || err == nil {
		t.Fatalf("read %v, err %v", read, err)
	}
	for _, want := range []string{"settings.conf:4: it is indented, but " +
		"under no block", "settings.conf:6: \"child\" is not a setting " +
		"of beta; its settings are other"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("missing %q in:\n%v", want, err)
		}
	}
	if lines := strings.Split(err.Error(), "\n"); len(lines) != bounced {
		t.Errorf("%d lines were named, not %d:\n%v", len(lines), bounced,
			err)
	}
}

// bounced is how many lines of that file are named. It's defined.
const bounced = 2

// Anti-test: a comment or a blank line in a block does not close it.
func TestACommentDoesNotCloseTheBlock(t *testing.T) {
	file := "alpha a\n# a comment\n\n  # indented\n  child 1\n"
	read, err := config.Read(fileName, []byte(file), twoBlocks()...)
	if err != nil || !slices.Equal(flat(read), []string{"alpha", "child"}) {
		t.Errorf("read %v, err %v", read, err)
	}
}
