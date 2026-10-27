package pzm_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// toolCasesFile lists calls to the tools below, one per line. Each line
// has the tool's key and then each word as a Go string. toolGoldenFile
// holds what each tool says, in order.
const (
	toolCasesFile  = "testdata/tools-cases.txt"
	toolGoldenFile = "testdata/tools-golden.txt"
)

// reporting is a verb that writes down that it ran. It gives its name and
// its words. Then it gives the value of each flag in names.
type reporting struct {
	name  string
	names []string
}

// do is the verb's run. It writes one line.
func (say reporting) do(call pzm.Call) error {
	var line strings.Builder
	fmt.Fprintf(&line, "ran %s words=%q", say.name, call.Args())
	for _, each := range say.names {
		fmt.Fprintf(&line, " %s=%q", each, call.Value(each))
	}
	_, err := fmt.Fprintln(call.Out(), line.String())
	return err
}

// pairTool has two verbs that give the short flag -n to flags of
// different names. Hello also has --number as -u. Count has it as -n.
func pairTool() *pzm.Tool {
	tool := pzm.New("hi", "a tool with a short flag two verbs share",
		"", "", "", "")
	tool.Verb("hello", 'e', "says hello", "# hello", "ExampleHello",
		readsOnly, reporting{"hello", []string{"name", "number"}}.do,
		pzm.Value("name", 'n', "who to say hello to"),
		pzm.Value("number", 'u', "how many times"))
	tool.Verb("count", 'c', "counts", "# count", "ExampleCount",
		readsOnly, reporting{"count", []string{"number"}}.do,
		pzm.Value("number", 'n', "how many"))
	return tool
}

// clashTool has a verb with a flag whose short letter asks for another
// verb. Before the verb the tool would read that flag as the verb.
func clashTool() *pzm.Tool {
	tool := pzm.New("hi", "a tool whose flag asks for a verb", "", "", "",
		"")
	tool.Verb("delete", 'd', "deletes", "# delete", "ExampleDelete",
		deletes, reporting{"delete", nil}.do)
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", readsOnly,
		reporting{"look", []string{"deep"}}.do,
		pzm.Switch("deep", 'd', "looks deeper"))
	return tool
}

// askingTool has the verb delete with the short flag d. Its verb look
// declares flag. Whether that flag asks for a verb depends on its letter
// and its name.
func askingTool(flag pzm.Flag) *pzm.Tool {
	tool := pzm.New("hi", "a tool to check", "", "", "", "")
	tool.Verb("delete", 'd', "deletes", "# delete", "ExampleDelete",
		deletes, reporting{"delete", nil}.do)
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", readsOnly,
		reporting{"look", nil}.do, flag)
	return tool
}

// Anti-tests: Check refuses a verb's flag that asks for a verb, by its
// short letter or by its name. The verb may be another or its own. The
// refusal names the verb and the flag.
func TestAVerbFlagThatAsksForAVerbIsRefused(t *testing.T) {
	for _, each := range []struct {
		flag pzm.Flag
		says string
	}{
		{pzm.Switch("deep", 'd', "deeper"), "verb look: flag --deep has " +
			"the name or short flag that asks for verb delete"},
		{pzm.Switch("long", 'l', "longer"), "verb look: flag --long has " +
			"the name or short flag that asks for verb look"},
		{pzm.Value("delete", verb.NoShort, "which"), "verb look: flag --delete has " +
			"the name or short flag that asks for verb delete"},
		{pzm.List("look", verb.NoShort, "which"), "verb look: flag --look has the " +
			"name or short flag that asks for verb look"},
	} {
		err := askingTool(each.flag).Check()
		if err == nil || !strings.Contains(err.Error(), each.says) {
			t.Errorf("Check said %v; want %q", err, each.says)
		}
	}
}

// Anti-test: a flag with a letter and a name of its own is not refused.
func TestAVerbFlagThatAsksForNoVerbIsKept(t *testing.T) {
	for _, each := range []pzm.Flag{pzm.Switch("deep", 'e', "deeper"),
		pzm.Value("from", verb.NoShort, "where from"),
		pzm.Switch("loud", 'L', "louder")} {
		if err := askingTool(each).Check(); err != nil {
			t.Errorf("Check said %v", err)
		}
	}
}

// toolsByKey are the tools the cases name.
func toolsByKey() map[string]func() *pzm.Tool {
	return map[string]func() *pzm.Tool{"pair": pairTool,
		"clash": clashTool}
}

// Every call in the cases file does what the golden file says.
func TestTheToolsAreHeldToTheGolden(t *testing.T) {
	cases, err := os.ReadFile(toolCasesFile)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(toolGoldenFile)
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for _, line := range strings.Split(string(cases), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, words, _ := strings.Cut(line, " ")
		tool := toolsByKey()[key]()
		args := wordsOf(t, words)
		got.WriteString(saidBy(tool, args))
	}
	if got.String() != string(want) {
		t.Errorf("the tools changed; diff %s against what this run "+
			"says:\n%s", toolGoldenFile, got.String())
	}
}
