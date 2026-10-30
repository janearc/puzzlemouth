package config_test

import (
	"slices"
	"strings"
	"testing"
)

const (
	// builderProjects is how many projects testdata/builder-config
	// opens. It's defined.
	builderProjects = 7
	// gameLints is how many lint lines testdata/game.game holds. It's
	// defined.
	gameLints = 10
	// gameFile and builderFile are the two copies of game's real files.
	// They are defined.
	gameFile    = "game.game"
	builderFile = "builder-config"
	// otherKeys is how many lines of game's other keys the test below
	// reads. It's defined.
	otherStack = 1
)

// The keys of game that the two files do not use are read as they are
// written, each with the words it takes.
func TestGamesOtherKeysAreRead(t *testing.T) {
	file := strings.Join([]string{
		"cover 80",
		"cover allow FILE a reason in several words",
		"sweep allow FILE KIND OTHER",
		"needs CONTACT a name and email",
		"stack NAME REF theirs",
		"words PATH with spaces",
		"omit PATH",
		"run mocks . :: go generate ./...",
		"project p",
		"\tset NAME a value with spaces",
	}, "\n") + "\n"
	got, err := readInto(t, gameDeclared{}, []byte(file))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Cover) != 2 || len(got.Stack) != otherStack ||
		!got.Stack[0].Theirs || len(got.Run) != 1 {
		t.Errorf("read %+v", got)
	}
	wantSet := []string{"NAME", "a", "value", "with", "spaces"}
	if len(got.Project) != 1 || !slices.Equal(got.Project[0].Set[0], wantSet) {
		t.Errorf("the project was read as %+v", got.Project)
	}
}
