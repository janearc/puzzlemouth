package config_test

import (
	"slices"
	"testing"
)

type documented struct {
	Name    string `config:"name,1" help:"the project's name"`
	Hidden  string
	Stack   []quiet           `config:"stack,2-3" help:"a member: name, ref, theirs"`
	Run     []quiet           `config:"run,2,tail" help:"a command and its tail"`
	Lint    [][]string        `config:"lint,1+" help:"a lint rule and its words"`
	Project []documentedBlock `config:"project,1" help:"a project's block"`
}

type documentedBlock struct {
	Name string     `opening:"the project's name"`
	Road string     `config:"road,1" help:"where it is kept"`
	Set  [][]string `config:"set,2+" help:"a name and its value"`
}

func TestSettingsAreInFieldOrderWithTheirWordsAndHelp(t *testing.T) {
	schema := buildSchema[documented](t)
	settings := schema.Settings()
	var keys []string
	words := map[string]string{}
	help := map[string]string{}
	for _, each := range settings {
		keys = append(keys, each.Key)
		words[each.Key] = each.Words
		help[each.Key] = each.Help
	}
	wantKeys := []string{"name", "stack", "run", "lint", "project"}
	if !slices.Equal(keys, wantKeys) {
		t.Fatalf("the keys were %v", keys)
	}
	wantWords := map[string]string{"name": "1", "stack": "2-3", "run": "2",
		"lint": "1+", "project": "1"}
	for key, want := range wantWords {
		if words[key] != want {
			t.Errorf("%s takes %q, not %q", key, words[key], want)
		}
	}
	if help["stack"] != "a member: name, ref, theirs" {
		t.Errorf("stack's help was %q", help["stack"])
	}
}

func TestSettingsSayWhichKeysTakeATailAndWhichOpenABlock(t *testing.T) {
	schema := buildSchema[documented](t)
	tails := map[string]bool{}
	blocks := map[string]bool{}
	for _, each := range schema.Settings() {
		tails[each.Key] = each.Tail
		blocks[each.Key] = each.Block
	}
	if !tails["run"] || tails["name"] || tails["project"] {
		t.Errorf("tails were %v", tails)
	}
	if !blocks["project"] || blocks["lint"] || blocks["run"] {
		t.Errorf("blocks were %v", blocks)
	}
}

func TestABlocksSettingsHoldItsInnerSettingsInFieldOrder(t *testing.T) {
	schema := buildSchema[documented](t)
	for _, each := range schema.Settings() {
		if each.Key != "project" {
			if each.In != nil {
				t.Errorf("%s holds %v and is no block", each.Key, each.In)
			}
			continue
		}
		var inner []string
		help := map[string]string{}
		for _, in := range each.In {
			inner = append(inner, in.Key)
			help[in.Key] = in.Help
		}
		if !slices.Equal(inner, []string{"road", "set"}) {
			t.Errorf("project holds %v", inner)
		}
		if help["road"] != "where it is kept" {
			t.Errorf("road's help was %q", help["road"])
		}
	}
}

func TestAnUntaggedFieldIsNotASetting(t *testing.T) {
	schema := buildSchema[documented](t)
	for _, each := range schema.Settings() {
		if each.Key == "Hidden" || each.Key == "hidden" || each.Key == "" {
			t.Errorf("the untagged field appeared as %+v", each)
		}
	}
}
