package config_test

import (
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// The lines a file is read from, in a name the errors carry.
const (
	// readName is the file's name in every error. It's defined.
	readName = "settings.game"
	// lineOne is a place in a file, counted from one. It's defined.
	lineOne = 1
	// lineTwo is a place in a file. It's defined.
	lineTwo = 2
	// lineThree is a place in a file. It's defined.
	lineThree = 3
	// lineFive is a place in a file. It's defined.
	lineFive = 5
	// httpPort is a listen line. It's defined.
	httpPort = 80
	// httpsPort is another listen line. It's defined.
	httpsPort = 443
	// wantPort is the number in the conversions file. It's defined.
	wantPort = 8080
	// keptPort is a default port the file does not change. It's defined.
	keptPort = 1
)

// where is how a refusal begins: the file, the line and the key.
func where(line int, key string) string {
	number := strconv.Itoa(line)
	return readName + ":" + number + ": " + key + ": "
}

// lines is a file of these lines, each ended.
func lines(each ...string) []byte {
	return []byte(strings.Join(each, "\n") + "\n")
}

// readInto reads the file into a copy of start, and gives it back with the
// error.
func readInto[File any](t *testing.T, start File, raw []byte) (File, error) {
	t.Helper()
	schema := buildSchema[File](t)
	into := start
	err := schema.Read(readName, raw, &into)
	return into, err
}

// refusedAt checks that err holds the refusal's beginning: file, line and
// key.
func refusedAt(t *testing.T, err error, line int, key string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the file was read; wanted a refusal at %s", where(line, key))
	}
	text := err.Error()
	begins := where(line, key)
	if !strings.Contains(text, begins) {
		t.Errorf("wanted a refusal beginning %q, got:\n%s", begins, text)
	}
}

type scalarFile struct {
	Name string `config:"name,1" help:"the name"`
	Home string `config:"home,1" help:"where it lives"`
}

func TestAScalarKeyOnceIsRead(t *testing.T) {
	got, err := readInto(t, scalarFile{}, lines("name game"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "game" || got.Home != "" {
		t.Errorf("read %+v", got)
	}
}

func TestAScalarKeyTwiceIsRefusedAtTheSecondLine(t *testing.T) {
	_, err := readInto(t, scalarFile{}, lines("name game", "name other"))
	refusedAt(t, err, lineTwo, "name")
}

type sliceFile struct {
	Words [][]string `config:"words,1+" help:"a line of words"`
}

func TestASliceKeyManyTimesIsKeptInFileOrder(t *testing.T) {
	file := lines("words one two", "# between", "words three", "words four five six")
	got, err := readInto(t, sliceFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"one", "two"}, {"three"}, {"four", "five", "six"}}
	if !slices.EqualFunc(got.Words, want, slices.Equal) {
		t.Errorf("read %v", got.Words)
	}
}

type conversions struct {
	Name   string   `config:"name,1" help:"one word"`
	Author string   `config:"author,1+" help:"the rest of the line"`
	Port   int      `config:"port,1" help:"a whole number"`
	Tags   []string `config:"tags,1+" help:"the words"`
	Ports  []int    `config:"listen,1" help:"a whole number on each of many lines"`
}

func TestStringIntAndStringSliceAreConverted(t *testing.T) {
	file := lines("name game", "author Ada  Byron   King", "port 8080",
		"tags a b c", "listen 80", "listen 443")
	got, err := readInto(t, conversions{}, file)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "game" || got.Author != "Ada Byron King" ||
		got.Port != wantPort {
		t.Errorf("read %+v", got)
	}
	if !slices.Equal(got.Tags, []string{"a", "b", "c"}) {
		t.Errorf("tags were %v", got.Tags)
	}
	if !slices.Equal(got.Ports, []int{httpPort, httpsPort}) {
		t.Errorf("ports were %v", got.Ports)
	}
}

func TestAnIntThatIsNotAWholeNumberIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto(t, conversions{}, lines("name game", "port eighty"))
	refusedAt(t, err, lineTwo, "port")
}

func TestAStringSliceKeyOnTwoLinesIsRefusedAtTheSecond(t *testing.T) {
	_, err := readInto(t, conversions{}, lines("tags a b", "tags c"))
	refusedAt(t, err, lineTwo, "tags")
}

// seeing keeps the line it was given, and refuses the word "bad".
type seeing struct {
	Got config.Line
}

var errBadWord = errors.New("bad is not a word here")

func (each *seeing) SetLine(line config.Line) error {
	each.Got = line
	if slices.Contains(line.Words, "bad") {
		return errBadWord
	}
	return nil
}

type seeingFile struct {
	Run   []seeing `config:"run,2,tail" help:"a command and its tail"`
	Lint  []seeing `config:"lint,1+" help:"a rule"`
	Width seeing   `config:"width,1" help:"on one line at most"`
}

func TestALineSetterIsCalledWithTheRightWordsTailAndLine(t *testing.T) {
	file := lines("# a comment", "", "run mocks . :: go generate ./...",
		"", "lint width 80")
	got, err := readInto(t, seeingFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]config.Line{}
	for _, each := range got.Run {
		byKey["run"] = each.Got
	}
	for _, each := range got.Lint {
		byKey["lint"] = each.Got
	}
	run := byKey["run"]
	if !slices.Equal(run.Words, []string{"mocks", "."}) ||
		run.Tail != "go generate ./..." || run.At != lineThree {
		t.Errorf("run was given %+v", run)
	}
	lint := byKey["lint"]
	if !slices.Equal(lint.Words, []string{"width", "80"}) ||
		lint.Tail != "" || lint.At != lineFive {
		t.Errorf("lint was given %+v", lint)
	}
}

func TestALineSettersErrorIsReportedAtItsLine(t *testing.T) {
	_, err := readInto(t, seeingFile{}, lines("lint fine", "lint bad"))
	refusedAt(t, err, lineTwo, "lint")
	text := err.Error()
	if !strings.Contains(text, errBadWord.Error()) {
		t.Errorf("the line setter's reason is not in: %v", text)
	}
}

func TestALineSetterScalarKeyTwiceIsRefusedAtTheSecondLine(t *testing.T) {
	_, err := readInto(t, seeingFile{}, lines("width 1", "width 2"))
	refusedAt(t, err, lineTwo, "width")
}

func TestDefaultsAreKeptWhenAKeyIsAbsent(t *testing.T) {
	start := scalarFile{Name: "default", Home: "nowhere"}
	got, err := readInto(t, start, lines("name set"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "set" || got.Home != "nowhere" {
		t.Errorf("read %+v", got)
	}
}

func TestAnEmptyFileKeepsEveryDefault(t *testing.T) {
	start := conversions{Name: "n", Port: keptPort, Tags: []string{"kept"}}
	got, err := readInto(t, start, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "n" || got.Port != keptPort || !slices.Equal(got.Tags, start.Tags) {
		t.Errorf("read %+v", got)
	}
}

func TestASliceKeyInTheFileReplacesItsDefault(t *testing.T) {
	start := sliceFile{Words: [][]string{{"default"}}}
	got, err := readInto(t, start, lines("words from the file"))
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"from", "the", "file"}}
	if !slices.EqualFunc(got.Words, want, slices.Equal) {
		t.Errorf("read %v", got.Words)
	}
}

func TestAnUnknownKeyIsRefusedAtItsLine(t *testing.T) {
	_, err := readInto(t, scalarFile{}, lines("name game", "bogus 1"))
	refusedAt(t, err, lineTwo, "bogus")
}

type rangesFile struct {
	Exact   string `config:"exact,2" help:"exactly two"`
	Between string `config:"between,2-3" help:"two or three"`
	Least   string `config:"least,2+" help:"two or more"`
}

func TestWordsAreHeldToTheirRange(t *testing.T) {
	taken("exact a b", t)
	taken("between a b", t)
	taken("between a b c", t)
	taken("least a b", t)
	taken("least a b c d e", t)
	refusedForWords("exact a", "exact", t)
	refusedForWords("exact a b c", "exact", t)
	refusedForWords("between a", "between", t)
	refusedForWords("between a b c d", "between", t)
	refusedForWords("least a", "least", t)
}

// taken checks that the line is read under rangesFile.
func taken(line string, t *testing.T) {
	t.Helper()
	_, err := readInto(t, rangesFile{}, lines(line))
	if err != nil {
		t.Errorf("%q was refused: %v", line, err)
	}
}

// refusedForWords checks that the line is refused under rangesFile.
func refusedForWords(line, key string, t *testing.T) {
	t.Helper()
	_, err := readInto(t, rangesFile{}, lines(line))
	refusedAt(t, err, lineOne, key)
}

type tailsFile struct {
	Name string   `config:"name,1" help:"takes no tail"`
	Run  []seeing `config:"run,1,tail" help:"takes a tail"`
}

func TestATailIsHandedToTheLineSetterTrimmed(t *testing.T) {
	got, err := readInto(t, tailsFile{}, lines("run go ::    make all :: twice   "))
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range got.Run {
		if each.Got.Tail != "make all :: twice" {
			t.Errorf("the tail was %q", each.Got.Tail)
		}
	}
}

func TestAKeyThatTakesATailRefusesALineWithNone(t *testing.T) {
	_, err := readInto(t, tailsFile{}, lines("run go"))
	refusedAt(t, err, lineOne, "run")
}

func TestAKeyThatTakesATailRefusesAnEmptyOne(t *testing.T) {
	_, err := readInto(t, tailsFile{}, lines("run go ::"))
	refusedAt(t, err, lineOne, "run")
}

func TestAKeyThatTakesNoTailRefusesOne(t *testing.T) {
	_, err := readInto(t, tailsFile{}, lines("name game :: and a tail"))
	refusedAt(t, err, lineOne, "name")
}

type project struct {
	Name string     `opening:"the project's name"`
	Road string     `config:"road,1" help:"where it is kept"`
	Dist string     `config:"dist,1" help:"where it is published"`
	Set  [][]string `config:"set,2+" help:"a name and its value"`
}

type projectFile struct {
	Name    string    `config:"name,1" help:"the name"`
	Project []project `config:"project,1" help:"a project's block"`
}

func projectsByName(each []project) map[string]project {
	byName := map[string]project{}
	for _, one := range each {
		byName[one.Name] = one
	}
	return byName
}

func TestABlockIsFilledIntoAStructSliceInFileOrder(t *testing.T) {
	file := lines("name game", "project alpha", "  road ~/alpha.git",
		"project beta", "  road ~/beta.git", "  dist https://example.invalid/beta",
		"  set A one", "  set B two words", "name again")
	_, err := readInto(t, projectFile{}, file)
	if err == nil {
		t.Fatal("name twice should have been refused")
	}
	file = lines("name game", "project alpha", "  road ~/alpha.git",
		"project beta", "  road ~/beta.git", "  dist https://example.invalid/beta",
		"  set A one", "  set B two words")
	got, err := readInto(t, projectFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, each := range got.Project {
		order = append(order, each.Name)
	}
	if !slices.Equal(order, []string{"alpha", "beta"}) {
		t.Errorf("the projects came in the order %v", order)
	}
	byName := projectsByName(got.Project)
	alpha := byName["alpha"]
	beta := byName["beta"]
	if alpha.Road != "~/alpha.git" || alpha.Dist != "" || alpha.Set != nil {
		t.Errorf("alpha was read as %+v", alpha)
	}
	wantSet := [][]string{{"A", "one"}, {"B", "two", "words"}}
	if beta.Road != "~/beta.git" || beta.Dist != "https://example.invalid/beta" ||
		!slices.EqualFunc(beta.Set, wantSet, slices.Equal) {
		t.Errorf("beta was read as %+v", beta)
	}
}

func TestATabIndentedBlockIsRead(t *testing.T) {
	got, err := readInto(t, projectFile{}, lines("project alpha", "\troad ~/a.git",
		"\tdist d"))
	if err != nil {
		t.Fatal(err)
	}
	byName := projectsByName(got.Project)
	if byName["alpha"].Road != "~/a.git" || byName["alpha"].Dist != "d" {
		t.Errorf("read %+v", got.Project)
	}
}

func TestTheNextBlockMayChooseTabsOrSpacesAgain(t *testing.T) {
	file := lines("project alpha", "\troad ~/a.git", "project beta", "    road ~/b.git")
	_, err := readInto(t, projectFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
}

func TestATabMixedWithSpacesInOneIndentIsRefused(t *testing.T) {
	_, err := readInto(t, projectFile{}, lines("project alpha", " \troad ~/a.git"))
	refusedAt(t, err, lineTwo, "road")
}

func TestALineThatChangesTheKindOfIndentInsideItsBlockIsRefused(t *testing.T) {
	file := lines("project alpha", "\troad ~/a.git", "  dist d")
	_, err := readInto(t, projectFile{}, file)
	refusedAt(t, err, lineThree, "dist")
}

func TestAnIndentedLineUnderNoBlockIsRefused(t *testing.T) {
	_, err := readInto(t, projectFile{}, lines("name game", "  road ~/a.git"))
	refusedAt(t, err, lineTwo, "road")
}

func TestAKeyOfTheFileInsideABlockIsRefused(t *testing.T) {
	_, err := readInto(t, projectFile{}, lines("project alpha", "  name game"))
	refusedAt(t, err, lineTwo, "name")
}

func TestAScalarKeyTwiceInOneBlockIsRefused(t *testing.T) {
	file := lines("project alpha", "  road a", "  road b")
	_, err := readInto(t, projectFile{}, file)
	refusedAt(t, err, lineThree, "road")
}

func TestARoadInTwoBlocksIsNotTwice(t *testing.T) {
	file := lines("project alpha", "  road a", "project beta", "  road b")
	_, err := readInto(t, projectFile{}, file)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEveryRefusalIsReportedTogetherNamedByFileLineAndKey(t *testing.T) {
	file := lines("name game", "bogus 1", "name again", "project alpha",
		"  road a b")
	_, err := readInto(t, projectFile{}, file)
	if err == nil {
		t.Fatal("the file was read")
	}
	text := err.Error()
	wanted := []string{where(lineTwo, "bogus"), where(lineThree, "name"),
		where(lineFive, "road")}
	for _, begins := range wanted {
		if !strings.Contains(text, begins) {
			t.Errorf("the refusals do not say %q:\n%s", begins, text)
		}
	}
}

func TestARefusedFileFillsNothing(t *testing.T) {
	start := projectFile{Name: "default"}
	got, err := readInto(t, start, lines("name set", "project alpha",
		"  road a", "bogus 1"))
	if err == nil {
		t.Fatal("the file was read")
	}
	if got.Name != "default" || got.Project != nil {
		t.Errorf("a refused file left %+v", got)
	}
}

func TestLineSettersRefusalsAreReportedWithTheOthers(t *testing.T) {
	_, err := readInto(t, seeingFile{}, lines("lint bad", "bogus", "lint bad"))
	refusedAt(t, err, lineOne, "lint")
	refusedAt(t, err, lineTwo, "bogus")
	refusedAt(t, err, lineThree, "lint")
}

type openedBySetter struct {
	Name seeing `opening:"read by a line setter"`
	Road string `config:"road,1" help:"where"`
}

type openedFile struct {
	Project []openedBySetter `config:"project,1+" help:"a block"`
}

func TestABlocksOpeningLineIsHandedToItsLineSetter(t *testing.T) {
	got, err := readInto(t, openedFile{}, lines("", "project alpha beta", "  road a"))
	if err != nil {
		t.Fatal(err)
	}
	for _, each := range got.Project {
		line := each.Name.Got
		if !slices.Equal(line.Words, []string{"alpha", "beta"}) ||
			line.At != lineTwo {
			t.Errorf("the opening line was handed %+v", line)
		}
	}
}
