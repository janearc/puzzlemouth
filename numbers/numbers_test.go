package numbers_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/numbers"
)

// sound is a numbers file that conforms: one measured, one picked.
const sound = `numbers:
  - name: continuous-speed
    value: 6.84
    unit: m s-1 m-1/2
    says: the centreline speed over the root of the height
    kind: measured
    source:
      key: nbsir-79-1910
      where: table 1, pdf page 35, row v/z^1/2, weighted average
      page: 35
      text: "6.84"
  - name: deepest
    value: 2
    says: how many calls may stand inside one another
    kind: picked
    by: ada, 2026-10-18
`

func TestASoundFileIsReadByName(test *testing.T) {
	read, err := numbers.Read("sound.yaml", []byte(sound))
	if err != nil {
		test.Fatal(err)
	}
	speed, err := read.Get("continuous-speed")
	if err != nil {
		test.Fatal(err)
	}
	from := speed.From()
	said := fmt.Sprint(speed.Value(), " ", speed.Unit(), " ",
		speed.Kind(), " ", from.Key(), " ", from.Page(), " ", from.Text())
	if said != "6.84 m s-1 m-1/2 measured nbsir-79-1910 35 6.84" {
		test.Errorf("the measured number: %s", said)
	}
	deepest, err := read.Get("deepest")
	if err != nil || deepest.By() != "ada, 2026-10-18" ||
		deepest.Kind() != numbers.Picked {
		test.Errorf("the picked number: %+v, %v", deepest, err)
	}
	var names []string
	for _, each := range read.All() {
		names = append(names, each.Name())
	}
	if strings.Join(names, " ") != "continuous-speed deepest" {
		test.Errorf("in the order written: %v", names)
	}
}

func TestANumberNotInTheFileIsRefusedByName(test *testing.T) {
	read, err := numbers.Read("sound.yaml", []byte(sound))
	if err != nil {
		test.Fatal(err)
	}
	if _, err := read.Get("widest"); err == nil ||
		!strings.Contains(err.Error(), "sound.yaml holds no number "+
			"called widest") {
		test.Errorf("a number not there: %v", err)
	}
}

func TestAFileThatDoesNotConformIsRefusedWhole(test *testing.T) {
	for why, bad := range map[string]string{
		"a key nobody knows": strings.ReplaceAll(sound, "    by:",
			"    colour: red\n    by:"),
		"a measured number with no source": strings.ReplaceAll(sound,
			"    source:\n      key: nbsir-79-1910\n", "    note:\n"+
				"      key: nbsir-79-1910\n"),
		"a picked number with no picker": strings.ReplaceAll(sound,
			"    by: ada, 2026-10-18\n", ""),
		"a kind that is neither": strings.ReplaceAll(sound,
			"kind: picked", "kind: guessed"),
		"a name written twice": strings.ReplaceAll(sound,
			"name: deepest", "name: continuous-speed"),
		"a number with no value": strings.ReplaceAll(sound,
			"    value: 2\n", ""),
	} {
		if _, err := numbers.Read("bad.yaml", []byte(bad)); err == nil {
			test.Errorf("%s was taken", why)
		}
	}
}
