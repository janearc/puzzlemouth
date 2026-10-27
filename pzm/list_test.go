package pzm_test

import (
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

const (
	// fewestItems is the fewest items a list the property gives has.
	// It's defined.
	fewestItems = 1
	// mostItems is the most. It's defined.
	mostItems = 4
	// firstPlace is the first place in a list. It's defined.
	firstPlace = 0
)

// itemChars are what an item is made of. It has no comma, and some of
// its items begin with a hyphen.
const itemChars = "ab-_. 1"

// genItems draws the items of a list. None is empty.
func genItems(t *rapid.T) []string {
	chars := rapid.SampledFrom([]rune(itemChars))
	item := rapid.StringOfN(chars, fewestItems, mostItems, mostItems)
	return rapid.SliceOfN(item, fewestItems, mostItems).Draw(t, "items")
}

// The verb gets a list's items back as they were, in order. The value is
// the items joined by commas, and the flag may be spelled either way.
func TestAListGivesBackItsItems(t *testing.T) {
	rapid.Check(t, checkItemsBack)
}

// checkItemsBack is one draw of that property.
func checkItemsBack(t *rapid.T) {
	items := genItems(t)
	joined := strings.Join(items, ",")
	spellings := [][]string{{"--tags=" + joined}, {"--tags", joined}}
	spelled := rapid.SampledFrom(spellings).Draw(t, "spelling")
	tool, seen := importer()
	code, _, errs := run(tool, append([]string{"import-agent"}, spelled...)...)
	if code != ranCode || !slices.Equal(seen.tags, items) {
		t.Fatalf("%q: code %d, errs %q, tags %q", spelled, code, errs,
			seen.tags)
	}
}

// Anti-test: an empty item refuses the call wherever it is in the list.
// The refusal says so and no verb runs.
func TestAListWithAnEmptyItemIsRefused(t *testing.T) {
	rapid.Check(t, checkEmptyItem)
}

// checkEmptyItem is one draw of that property.
func checkEmptyItem(t *rapid.T) {
	items := genItems(t)
	at := rapid.IntRange(firstPlace, len(items)).Draw(t, "at")
	holed := slices.Insert(slices.Clone(items), at, "")
	tool, seen := importer()
	code, _, errs := run(tool, "import-agent", "--tags="+
		strings.Join(holed, ","))
	if code != refusedCode || !strings.Contains(errs, "has an empty item") ||
		seen.args != nil {
		t.Fatalf("%q: code %d, errs %q, heard %+v", holed, code, errs, seen)
	}
}

// Anti-test: a list given twice is refused, as any flag is. No verb
// runs.
func TestAListGivenTwiceIsRefused(t *testing.T) {
	tool, seen := importer()
	code, _, errs := run(tool, "import-agent", "--tags", "a", "--tags", "b")
	if code != refusedCode || !strings.Contains(errs, "given twice") ||
		seen.args != nil {
		t.Errorf("code %d, errs %q, heard %+v", code, errs, seen)
	}
}
