package config_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

func TestReadingIntoNothingIsRefused(t *testing.T) {
	schema := buildSchema[scalarFile](t)
	err := schema.Read(readName, lines("name game"), nil)
	if err == nil || !strings.Contains(err.Error(), readName) {
		t.Errorf("reading into nothing gave %v", err)
	}
}

func TestTheZeroSchemaRefusesEveryLine(t *testing.T) {
	var schema config.Schema[scalarFile]
	var into scalarFile
	err := schema.Read(readName, lines("name game"), &into)
	refusedAt(t, err, lineOne, "name")
	if len(schema.Settings()) != 0 {
		t.Error("the zero schema holds settings")
	}
}

func TestALineWithOnlyADividerAndAnOddSpaceHasNoKey(t *testing.T) {
	_, err := readInto(t, scalarFile{}, lines("  :: a tail"))
	if err == nil || !strings.Contains(err.Error(), readName+":1: ") ||
		!strings.Contains(err.Error(), "no key") {
		t.Errorf("a line with no key gave %v", err)
	}
}

func TestAnEqualsSignAfterTheKeyReadsAsNothing(t *testing.T) {
	got, err := readInto(t, scalarFile{}, lines("name = game"))
	if err != nil || got.Name != "game" {
		t.Errorf("read %+v, %v", got, err)
	}
}

func TestAnEmptyTailAfterTheLastWordIsAMissingTail(t *testing.T) {
	_, err := readInto(t, tailsFile{}, lines("run go ::"))
	refusedAt(t, err, lineOne, "run")
}

func TestADoubleColonIsAWordWhereNoTailIsTaken(t *testing.T) {
	got, err := readInto(t, sliceFile{}, lines("words listen ::1 ::"))
	if err != nil || len(got.Words) != 1 || len(got.Words[0]) != lineThree {
		t.Errorf("read %+v, %v", got, err)
	}
}

func TestAWordRangeIsSaidInTheRefusal(t *testing.T) {
	for _, line := range []string{"exact a", "between a", "least a"} {
		_, err := readInto(t, rangesFile{}, lines(line))
		text := err.Error()
		if !strings.Contains(text, "word") {
			t.Errorf("%q: %v", line, err)
		}
	}
	_, err := readInto(t, rangesFile{}, lines("exact a b c"))
	if !strings.Contains(err.Error(), "takes 2 words") {
		t.Errorf("the range was not said: %v", err)
	}
	_, err = readInto(t, rangesFile{}, lines("between a"))
	if !strings.Contains(err.Error(), "2 to 3 words") {
		t.Errorf("the range was not said: %v", err)
	}
	_, err = readInto(t, rangesFile{}, lines("least a"))
	if !strings.Contains(err.Error(), "at least 2 words") {
		t.Errorf("the range was not said: %v", err)
	}
}

type countedBlock struct {
	Port int    `opening:"a whole number"`
	Road string `config:"road,1" help:"where"`
}

type countedFile struct {
	Project []countedBlock `config:"project,1" help:"a block"`
}

func TestABlocksOpeningIntThatIsNotANumberIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto(t, countedFile{}, lines("project eighty", "  road a"))
	refusedAt(t, err, lineOne, "project")
}

type bareBlock struct {
	Road string `config:"road,1" help:"where"`
}

type bareFile struct {
	Project []bareBlock `config:"project,1" help:"a block with nowhere for its word"`
}

func TestABlockWithNoOpeningFieldIsRefusedByField(t *testing.T) {
	refusedByField[bareFile](t, "Project")
}

type floatSlice struct {
	Ratios []float64 `config:"ratios,1" help:"a slice of what is not read"`
}

func TestASliceOfATypeItCannotFillIsRefusedByField(t *testing.T) {
	refusedByField[floatSlice](t, "Ratios")
}

type intOpening struct {
	Count int    `opening:"a count"`
	Road  string `config:"road,1" help:"where"`
}

type intOpeningFile struct {
	Project []intOpening `config:"project,2" help:"two words for one int"`
}

func TestAnIntOpeningThatIsGivenTwoWordsIsRefused(t *testing.T) {
	refusedByField[intOpeningFile](t, "Project")
}

type blankOpening struct {
	Name string `opening:" "`
	Road string `config:"road,1" help:"where"`
}

type blankOpeningFile struct {
	Project []blankOpening `config:"project,1" help:"a block"`
}

func TestAnOpeningWithNoHelpIsRefusedByField(t *testing.T) {
	refusedByField[blankOpeningFile](t, "Name")
}

type floatOpening struct {
	Name float64 `opening:"a float"`
	Road string  `config:"road,1" help:"where"`
}

type floatOpeningFile struct {
	Project []floatOpening `config:"project,1" help:"a block"`
}

func TestAnOpeningTypeItCannotFillIsRefusedByField(t *testing.T) {
	refusedByField[floatOpeningFile](t, "Name")
}

type hidden struct {
	secret string `config:"secret,1" help:"nobody can set this"`
}

func TestATaggedFieldThatIsNotExportedIsRefusedByField(t *testing.T) {
	refusedByField[hidden](t, "secret")
	if (hidden{secret: "kept"}).secret != "kept" {
		t.Error("the field holds what it is given")
	}
}

type bothTags struct {
	Both string `config:"both,1" opening:"and an opening" help:"both"`
}

func TestAFieldWithBothTagsIsRefusedByField(t *testing.T) {
	refusedByField[bothTags](t, "Both")
}

type tagTooLong struct {
	Long string `config:"long,1,tail,more" help:"too many parts"`
}

type tagNotTail struct {
	Odd string `config:"odd,1,trail" help:"a third part that is not tail"`
}

func TestATagWithTheWrongPartsIsRefusedByField(t *testing.T) {
	refusedByField[tagTooLong](t, "Long")
	refusedByField[tagNotTail](t, "Odd")
}

type openAndRanged struct {
	Both string `config:"both,1-2+" help:"two kinds of open end"`
}

func TestARangeThatIsOpenAndRangedIsRefusedByField(t *testing.T) {
	refusedByField[openAndRanged](t, "Both")
}

type blockWithTail struct {
	Project []blockOfTail `config:"project,1,tail" help:"a block with a tail"`
}

type blockOfTail struct {
	Name string `opening:"its name"`
	Road string `config:"road,1" help:"where"`
}

func TestABlockWithATailIsRefusedByField(t *testing.T) {
	refusedByField[blockWithTail](t, "Project")
}

type setterCountsInts struct {
	Many []quiet `config:"many,0+" help:"zero or more, each a line"`
}

func TestARangeFromZeroIsAllowed(t *testing.T) {
	buildSchema[setterCountsInts](t)
}

func TestSettingsAreACopyTheCallerMayChange(t *testing.T) {
	schema := buildSchema[documented](t)
	for _, each := range schema.Settings() {
		each.Key = "changed"
		if len(each.In) > 0 {
			each.In[0].Key = "changed"
		}
	}
	for _, each := range schema.Settings() {
		if each.Key == "changed" || len(each.In) > 0 &&
			each.In[0].Key == "changed" {
			t.Errorf("the schema was changed through Settings: %+v", each)
		}
	}
}
