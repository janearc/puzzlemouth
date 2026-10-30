package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// These stand in for who.Signer and verb.Width, so that pzm/settings.go
// can be seen as it would read: a declared type, one Read, no loop over
// entries and no seen map.

const (
	// onlyWord is the place of the one word of each of pzm's lines. It's
	// defined.
	onlyWord = 0
	// setWidth is the width the file sets. It's defined.
	setWidth = 80
	// defaultWidth is the width held before the file is read. It's
	// defined.
	defaultWidth = 100
	// smallestWidth is the least a width may be. It's defined.
	smallestWidth = 1
)

// signerStandIn is "signer WHO :: NAME", as who.Signer would read it.
type signerStandIn struct {
	Who, Name string
}

func (signer *signerStandIn) SetLine(line config.Line) error {
	signer.Who = line.Words[onlyWord]
	signer.Name = line.Tail
	return nil
}

// widthStandIn is "help-width N", as verb.Width would read it.
type widthStandIn int

var errWidth = errors.New("a width is a whole number above zero")

func (width *widthStandIn) SetLine(line config.Line) error {
	got, err := strconv.Atoi(line.Words[onlyWord])
	if err != nil || got < smallestWidth {
		return errWidth
	}
	*width = widthStandIn(got)
	return nil
}

// pzmFile is pzm's own settings, as design.md has them.
type pzmFile struct {
	Signers []signerStandIn `config:"signer,1,tail" help:"a person whose signature on a build --who vouches for"`
	Width   widthStandIn    `config:"help-width,1" help:"the most characters a line of help may have"`
}

// pzmSettingsFile is a settings file under a home that is a temporary
// directory, the way a test of pzm's settings sets one up.
func pzmSettingsFile(t *testing.T, text []byte) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "pzm.settings")
	err := os.WriteFile(path, text, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPzmsDeclarationIsMade(t *testing.T) {
	buildSchema[pzmFile](t)
}

func TestPzmsSettingsAreReadFromARealFile(t *testing.T) {
	text := lines("# who may sign", "signer ada@example.invalid :: Ada Lovelace",
		"signer grace@example.invalid :: Grace Hopper", "help-width 80")
	path := pzmSettingsFile(t, text)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	schema := buildSchema[pzmFile](t)
	var got pzmFile
	err = schema.Read(path, raw, &got)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != setWidth {
		t.Errorf("the width was %d", got.Width)
	}
	names := map[string]string{}
	for _, each := range got.Signers {
		names[each.Who] = each.Name
	}
	if names["ada@example.invalid"] != "Ada Lovelace" ||
		names["grace@example.invalid"] != "Grace Hopper" {
		t.Errorf("the signers were %v", names)
	}
}

func TestPzmsWidthKeepsItsDefaultWhenTheFileSaysNothing(t *testing.T) {
	start := pzmFile{Width: defaultWidth}
	got, err := readInto(t, start, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Width != defaultWidth {
		t.Errorf("the width was %d", got.Width)
	}
}

func TestPzmsSignerWithoutATailIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto(t, pzmFile{}, lines("help-width 80", "signer ada@example.invalid"))
	refusedAt(t, err, lineTwo, "signer")
}

func TestPzmsWidthThatIsNotAWidthIsRefusedWithTheSetterReason(t *testing.T) {
	_, err := readInto(t, pzmFile{}, lines("help-width wide"))
	refusedAt(t, err, lineOne, "help-width")
}

func TestPzmsHelpWidthTwiceIsRefusedAtTheSecondLine(t *testing.T) {
	_, err := readInto(t, pzmFile{}, lines("help-width 80", "help-width 90"))
	refusedAt(t, err, lineTwo, "help-width")
}

func TestPzmsSettingsListSignerAndHelpWidth(t *testing.T) {
	schema := buildSchema[pzmFile](t)
	var keys []string
	tails := map[string]bool{}
	for _, each := range schema.Settings() {
		keys = append(keys, each.Key)
		tails[each.Key] = each.Tail
	}
	if !slices.Equal(keys, []string{"signer", "help-width"}) {
		t.Errorf("the keys were %v", keys)
	}
	if !tails["signer"] || tails["help-width"] {
		t.Errorf("tails were %v", tails)
	}
}
