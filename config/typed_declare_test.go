package config_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// A word LineSetter that keeps nothing. It stands for any type that reads
// its own line.
type quiet struct{}

func (*quiet) SetLine(config.Line) error { return nil }

// buildSchema is the schema of File, or a failed test.
func buildSchema[File any](t *testing.T) *config.Schema[File] {
	t.Helper()
	schema, err := config.NewSchema[File]()
	if err != nil {
		t.Fatalf("the declaration was refused: %v", err)
	}
	if schema == nil {
		t.Fatal("the declaration was taken and no schema was returned")
	}
	return schema
}

// refusedByField checks that File is refused, that no schema comes back
// with the refusal, and that the refusal names the field, as "field
// NAME:".
func refusedByField[File any](t *testing.T, field string) {
	t.Helper()
	schema, err := config.NewSchema[File]()
	if err == nil {
		t.Fatalf("a declaration with a bad %s was accepted", field)
	}
	if schema != nil {
		t.Errorf("a refusal for %s came with a schema", field)
	}
	text := err.Error()
	named := "field " + field + ":"
	if !strings.Contains(text, named) {
		t.Errorf("the refusal does not say %q: %v", named, text)
	}
}

// A good schema: scalars, a slice, a block, a line setter, a tail.
type goodFile struct {
	Name    string      `config:"name,1" help:"the name"`
	Port    int         `config:"port,1" help:"the port"`
	Author  []string    `config:"author,1+" help:"who wrote it"`
	Words   [][]string  `config:"words,1+" help:"a line of words"`
	Lint    []quiet     `config:"lint,1+" help:"a lint rule"`
	Run     []quiet     `config:"run,2,tail" help:"a command and its tail"`
	Project []goodBlock `config:"project,1" help:"a project's block"`
	Skipped string
}

type goodBlock struct {
	Name string     `opening:"the project's name"`
	Road string     `config:"road,1" help:"where it is kept"`
	Set  [][]string `config:"set,2+" help:"a name and its value"`
}

func TestAGoodSchemaIsMade(t *testing.T) {
	buildSchema[goodFile](t)
}

type shoutedKey struct {
	Shout string `config:"Shout,1" help:"capitals"`
}

type underscoredKey struct {
	Snake string `config:"snake_case,1" help:"an underscore"`
}

type hyphenLedKey struct {
	Lead string `config:"-lead,1" help:"a hyphen first"`
}

type emptyKey struct {
	Nothing string `config:",1" help:"no key"`
}

func TestAKeyThatIsNotLowercaseWordsJoinedByHyphensIsRefusedByField(t *testing.T) {
	refusedByField[shoutedKey](t, "Shout")
	refusedByField[underscoredKey](t, "Snake")
	refusedByField[hyphenLedKey](t, "Lead")
	refusedByField[emptyKey](t, "Nothing")
}

type keyTwice struct {
	First  string `config:"name,1" help:"the first"`
	Second string `config:"name,1" help:"the same key"`
}

type keyTwiceInABlock struct {
	Project []twiceInside `config:"project,1" help:"a block"`
}

type twiceInside struct {
	Name  string `opening:"its name"`
	Road  string `config:"road,1" help:"the first"`
	Other string `config:"road,1" help:"the same key"`
}

func TestAKeyUsedTwiceIsRefusedByTheSecondField(t *testing.T) {
	refusedByField[keyTwice](t, "Second")
}

func TestAKeyUsedTwiceInOneBlockIsRefusedByTheSecondField(t *testing.T) {
	refusedByField[keyTwiceInABlock](t, "Other")
}

type helpMissing struct {
	Bare string `config:"bare,1"`
}

type helpBlank struct {
	Blank string `config:"blank,1" help:"   "`
}

type helpTwoLines struct {
	Long string `config:"long,1" help:"first line\nsecond line"`
}

func TestMissingOrMultiLineHelpIsRefusedByField(t *testing.T) {
	refusedByField[helpMissing](t, "Bare")
	refusedByField[helpBlank](t, "Blank")
	refusedByField[helpTwoLines](t, "Long")
}

type rangeBackwards struct {
	Back string `config:"back,3-2" help:"most below fewest"`
}

type rangeNothing struct {
	Empty string `config:"empty,0" help:"no words and no tail"`
}

type rangeNotNumbers struct {
	Many string `config:"many,many" help:"not a number"`
}

type rangeOpenEnded struct {
	Open string `config:"open,3-" help:"most left out"`
}

type rangeMissing struct {
	Lone string `config:"lone" help:"no range at all"`
}

func TestARangeThatCannotBeMetIsRefusedByField(t *testing.T) {
	refusedByField[rangeBackwards](t, "Back")
	refusedByField[rangeNothing](t, "Empty")
	refusedByField[rangeNotNumbers](t, "Many")
	refusedByField[rangeOpenEnded](t, "Open")
	refusedByField[rangeMissing](t, "Lone")
}

type tailOnly struct {
	Command quiet `config:"command,0,tail" help:"nothing but a tail"`
}

func TestNoWordsWithATailIsAllowed(t *testing.T) {
	buildSchema[tailOnly](t)
}

type intTakesTwo struct {
	Pair int `config:"pair,2" help:"two words for one int"`
}

type intTakesAtLeast struct {
	Some int `config:"some,1+" help:"any words for one int"`
}

type intTakesBetween struct {
	Few int `config:"few,1-2" help:"a range for one int"`
}

func TestARangeAScalarTypeCannotTakeIsRefusedByField(t *testing.T) {
	refusedByField[intTakesTwo](t, "Pair")
	refusedByField[intTakesAtLeast](t, "Some")
	refusedByField[intTakesBetween](t, "Few")
}

type floatField struct {
	Ratio float64 `config:"ratio,1" help:"floats are not read"`
}

type mapField struct {
	Table map[string]string `config:"table,1" help:"maps are not read"`
}

type pointerField struct {
	Maybe *string `config:"maybe,1" help:"pointers are not read"`
}

type untaggedElements struct {
	Rows []struct{ Seen int } `config:"rows,1" help:"a struct with no tags is no block"`
}

type boolField struct {
	Yes bool `config:"yes,1" help:"bools are not read"`
}

func TestAFieldTypeItCannotFillIsRefusedByField(t *testing.T) {
	refusedByField[floatField](t, "Ratio")
	refusedByField[mapField](t, "Table")
	refusedByField[pointerField](t, "Maybe")
	refusedByField[untaggedElements](t, "Rows")
	refusedByField[boolField](t, "Yes")
}

type blockInBlock struct {
	Outer []holdsBlock `config:"outer,1" help:"a block that holds a block"`
}

type holdsBlock struct {
	Name  string       `opening:"its name"`
	Inner []innerBlock `config:"inner,1" help:"a block in a block"`
}

type innerBlock struct {
	Name string `opening:"its name"`
	Deep string `config:"deep,1" help:"deeper than a file goes"`
}

func TestABlockInABlockIsRefusedByField(t *testing.T) {
	refusedByField[blockInBlock](t, "Inner")
}

type tailOnAString struct {
	Say string `config:"say,1,tail" help:"a string has nowhere to put a tail"`
}

func TestATailOnATypeThatIsNotALineSetterIsRefusedByField(t *testing.T) {
	refusedByField[tailOnAString](t, "Say")
}

type openingWordsLost struct {
	Project []noOpening `config:"project,1" help:"a block whose word has no field"`
}

type noOpening struct {
	Road string `config:"road,1" help:"where it is kept"`
}

type openingOutsideABlock struct {
	Name string `opening:"only a block opens"`
}

type openingTwice struct {
	Project []twoOpenings `config:"project,1" help:"a block with two openings"`
}

type twoOpenings struct {
	First  string `opening:"one"`
	Second string `opening:"two"`
	Road   string `config:"road,1" help:"where it is kept"`
}

func TestAnOpeningThatCannotWorkIsRefusedByField(t *testing.T) {
	refusedByField[openingWordsLost](t, "Project")
	refusedByField[openingOutsideABlock](t, "Name")
	refusedByField[openingTwice](t, "Second")
}

type twoBad struct {
	Shout string  `config:"Shout,1" help:"capitals"`
	Bare  string  `config:"bare,1"`
	Ratio float64 `config:"ratio,1" help:"floats are not read"`
}

func TestEveryMistakeInADeclarationIsReportedTogether(t *testing.T) {
	schema, err := config.NewSchema[twoBad]()
	if err == nil {
		t.Fatal("a declaration with three mistakes was accepted")
	}
	if schema != nil {
		t.Error("the refusal came with a schema")
	}
	text := err.Error()
	for _, field := range []string{"Shout", "Bare", "Ratio"} {
		named := "field " + field + ":"
		if !strings.Contains(text, named) {
			t.Errorf("the refusal does not say %q: %v", named, text)
		}
	}
}

func TestATypeThatIsNotAStructIsRefused(t *testing.T) {
	schema, err := config.NewSchema[int]()
	if err == nil {
		t.Fatal("an int was accepted as a file")
	}
	if schema != nil {
		t.Error("the refusal came with a schema")
	}
	text := err.Error()
	if !strings.Contains(text, "struct") {
		t.Errorf("the refusal does not say a file is a struct: %v", text)
	}
}
