package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/config"
)

// Game's own file, declared as a type. Game keeps this declaration, not
// pzm: it is here to show that the real files are read by the design.

const (
	// memberName is the place of a stack line's name. It's defined.
	memberName = 0
	// memberRef is the place of a stack line's ref. It's defined.
	memberRef = 1
	// memberTheirs is the place of a stack line's optional third word.
	// It's defined.
	memberTheirs = 2
	// ruleAt is the place of a lint line's rule. It's defined.
	ruleAt = 0
	// argsAt is where a lint line's arguments begin. It's defined.
	argsAt = 1
	// commandName is the place of a target's name. It's defined.
	commandName = 0
	// commandDir is the place of a target's directory. It's defined.
	commandDir = 1
	// targetsInGame is how many target lines testdata/game.game holds.
	// It's defined.
	targetsInGame = 3
)

// stackMember is "stack NAME REF [theirs]".
type stackMember struct {
	Name, Ref string
	Theirs    bool
}

var errTheirs = errors.New("the third word, when there is one, is theirs")

func (member *stackMember) SetLine(line config.Line) error {
	member.Name = line.Words[memberName]
	member.Ref = line.Words[memberRef]
	if len(line.Words) > memberTheirs {
		if line.Words[memberTheirs] != "theirs" {
			return errTheirs
		}
		member.Theirs = true
	}
	return nil
}

// lintLine is "lint RULE [ARGS...]".
type lintLine struct {
	Rule string
	Args []string
}

func (each *lintLine) SetLine(line config.Line) error {
	each.Rule = line.Words[ruleAt]
	each.Args = line.Words[argsAt:]
	return nil
}

// commandLine is "target NAME DIR :: COMMAND", and run likewise.
type commandLine struct {
	Name, Dir, Command string
}

func (each *commandLine) SetLine(line config.Line) error {
	each.Name = line.Words[commandName]
	each.Dir = line.Words[commandDir]
	each.Command = line.Tail
	return nil
}

// projectBlock is a project, and what its indented lines set.
type projectBlock struct {
	Name string     `opening:"the project's name"`
	Road string     `config:"road,1" help:"where its repository is kept"`
	Dist string     `config:"dist,1" help:"where it is published"`
	Set  [][]string `config:"set,2+" help:"a name and its value, kept for the build"`
}

// gameDeclared is every key game's files may set.
type gameDeclared struct {
	Name    string         `config:"name,1" help:"the project's name"`
	Author  []string       `config:"author,1+" help:"who wrote it"`
	Exclude []string       `config:"exclude,1+" help:"paths left out of a read"`
	Words   [][]string     `config:"words,1+" help:"a path and the words it may use"`
	Omit    [][]string     `config:"omit,1+" help:"a path left out"`
	Stack   []stackMember  `config:"stack,2-3" help:"a member: name, ref, theirs"`
	Lint    []lintLine     `config:"lint,1+" help:"a lint rule and its words"`
	Cover   [][]string     `config:"cover,1+" help:"the cover floor, or an allowance"`
	Sweep   [][]string     `config:"sweep,3+" help:"a sweep allowance"`
	Needs   [][]string     `config:"needs,2+" help:"a contact and why"`
	Rapid   [][]string     `config:"rapid,2" help:"a property test setting"`
	Target  []commandLine  `config:"target,2,tail" help:"a build target and its command"`
	Run     []commandLine  `config:"run,2,tail" help:"a run and its command"`
	Project []projectBlock `config:"project,1" help:"a project's block"`
}

// readTestdata is a real file of game's, read as game's declaration.
func readTestdata(t *testing.T, name string) gameDeclared {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	got, err := readInto(t, gameDeclared{}, raw)
	if err != nil {
		t.Fatalf("%s was refused:\n%v", name, err)
	}
	return got
}

func TestGamesDeclarationIsMade(t *testing.T) {
	buildSchema[gameDeclared](t)
}

func TestGamesOwnFileIsReadAsATypeWithItsLintLines(t *testing.T) {
	got := readTestdata(t, gameFile)
	if got.Name != "game" {
		t.Errorf("the name was %q", got.Name)
	}
	byRule := map[string][]string{}
	for _, each := range got.Lint {
		byRule[each.Rule] = each.Args
	}
	if len(got.Lint) != gameLints {
		t.Errorf("%d lint lines", len(got.Lint))
	}
	wantWords := []string{"hyperscaler", "enterprise-grade", "blazing-fast",
		"load-bearing"}
	if !slices.Equal(byRule["rows"], []string{"25"}) ||
		!slices.Equal(byRule["words"], wantWords) {
		t.Errorf("the lint lines were read as %v", byRule)
	}
}

func TestGamesOwnFileKeepsItsStackInOrderWithTheirRefs(t *testing.T) {
	got := readTestdata(t, gameFile)
	var names []string
	refs := map[string]string{}
	for _, each := range got.Stack {
		names = append(names, each.Name)
		refs[each.Name] = each.Ref
	}
	if !slices.Equal(names, []string{"puzzlemouth", "libninja", "okapi"}) {
		t.Errorf("the stack was %v", names)
	}
	if refs["okapi"] != "main" {
		t.Errorf("okapi is taken at %q", refs["okapi"])
	}
}

func TestGamesOwnFileKeepsEachTargetsCommandAsItsTail(t *testing.T) {
	got := readTestdata(t, gameFile)
	if len(got.Target) != targetsInGame {
		t.Fatalf("%d targets", len(got.Target))
	}
	for _, each := range got.Target {
		prefix := "go run go.uber.org/mock/mockgen@v0.6.0 "
		started := strings.HasPrefix(each.Command, prefix)
		if each.Name != "mocks" || each.Dir != "." || !started {
			t.Errorf("a target was read as %+v", each)
		}
	}
}

func TestABuildersConfigIsReadAsATypeWithItsProjects(t *testing.T) {
	got := readTestdata(t, builderFile)
	if len(got.Project) != builderProjects {
		t.Fatalf("%d projects", len(got.Project))
	}
	byName := projectsInBlocks(got.Project)
	roadOf := map[string]string{
		"alpha": "~/roots/alpha-road.git",
		"beta":  "~/roots/beta-road.git",
		"delta": "~/roots/delta.git",
	}
	for name, want := range roadOf {
		if byName[name].Road != want {
			t.Errorf("%s has the road %q, not %q", name, byName[name].Road, want)
		}
	}
	if byName["beta"].Dist != "https://example.invalid/beta.git" {
		t.Errorf("beta is published at %q", byName["beta"].Dist)
	}
	if byName["alpha"].Dist != "" {
		t.Errorf("alpha is published at %q", byName["alpha"].Dist)
	}
}

func TestABuildersConfigKeepsAProjectsSetWords(t *testing.T) {
	got := readTestdata(t, builderFile)
	byName := projectsInBlocks(got.Project)
	want := [][]string{{"SHELF_DIR", "/srv/shelf/sources"}}
	if !slices.EqualFunc(byName["delta"].Set, want, slices.Equal) {
		t.Errorf("delta set %v", byName["delta"].Set)
	}
}

func TestABuildersConfigKeepsItsAuthorAndExclude(t *testing.T) {
	got := readTestdata(t, builderFile)
	if !slices.Equal(got.Author, []string{"someone"}) ||
		!slices.Equal(got.Exclude, []string{"spec-docs/"}) {
		t.Errorf("author %v, exclude %v", got.Author, got.Exclude)
	}
}

func projectsInBlocks(each []projectBlock) map[string]projectBlock {
	byName := map[string]projectBlock{}
	for _, one := range each {
		byName[one.Name] = one
	}
	return byName
}

func TestGamesLineSettersRefusalIsReportedAtItsLine(t *testing.T) {
	_, err := readInto(t, gameDeclared{}, lines("name game", "stack puzzlemouth main mine"))
	refusedAt(t, err, lineTwo, "stack")
}

func TestGamesSettingsListEveryKeyInFieldOrder(t *testing.T) {
	schema := buildSchema[gameDeclared](t)
	var keys []string
	for _, each := range schema.Settings() {
		keys = append(keys, each.Key)
	}
	want := []string{"name", "author", "exclude", "words", "omit", "stack",
		"lint", "cover", "sweep", "needs", "rapid", "target", "run", "project"}
	if !slices.Equal(keys, want) {
		t.Errorf("the keys were %v", keys)
	}
}
