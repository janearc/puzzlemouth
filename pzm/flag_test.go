package pzm_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// heard is what the importer verb was given: its words and its flags.
type heard struct {
	args  []string
	from  string
	given bool
	loud  bool
	tags  []string
	roots string
	in    string
}

// importer is a tool with one verb that takes every kind of flag, a verb
// with none, and a flag of the tool's own; and what the first heard.
func importer() (*pzm.Tool, *heard) {
	seen := &heard{}
	tool := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	tool.Flags(pzm.Value("roots", 'r', "where the roots are"))
	tool.Verb("import-agent", 'i', "imports an agent", "# import-agent",
		"ExampleImportAgent", readsOnly, func(call pzm.Call) error {
			seen.args = call.Args()
			seen.from, seen.given = call.Value("from"), call.Given("from")
			seen.loud = call.On("loud")
			seen.tags = call.List("tags")
			seen.roots = call.Value("roots")
			if call.In() != nil {
				raw, _ := io.ReadAll(call.In())
				seen.in = string(raw)
			}
			return nil
		},
		pzm.Value("from", 'f', "the old home"),
		pzm.Switch("loud", 'l', "says more"),
		pzm.List("tags", verb.NoShort, "words to keep"))
	tool.Verb("plain", verb.NoShort, "takes no flags", "# plain", "ExamplePlain", readsOnly,
		func(pzm.Call) error { return nil })
	return tool, seen
}

// Flags and words mix, in any order, the way they are typed: the name
// first, then the flags, or the flags first.
func TestFlagsAndWordsMix(t *testing.T) {
	for _, asked := range [][]string{
		{"import-agent", "ada", "--from", "old", "--loud",
			"--tags", "a,b"},
		{"import-agent", "--from=old", "ada", "-l", "--tags=a,b"},
		{"import-agent", "-f", "old", "--tags", "a,b", "ada", "--loud"},
		{"--import-agent", "ada", "--from", "old", "-l", "--tags", "a,b"},
		{"-i", "ada", "-f", "old", "-l", "--tags", "a,b"},
	} {
		tool, seen := importer()
		code, _, errs := run(tool, asked...)
		if code != exitOK || strings.Join(seen.args, " ") != "ada" ||
			seen.from != "old" || !seen.given || !seen.loud ||
			strings.Join(seen.tags, "|") != "a|b" {
			t.Errorf("%v: code %d, errs %q, heard %+v", asked, code, errs,
				seen)
		}
	}
}

// A flag not given reads as nothing, and Given tells a value of nothing
// given from no value at all.
func TestAFlagNotGiven(t *testing.T) {
	tool, seen := importer()
	if code, _, errs := run(tool, "import-agent", "ada"); code != exitOK ||
		seen.from != "" || seen.given || seen.loud || seen.tags != nil {
		t.Errorf("code %d, errs %q, heard %+v", code, errs, seen)
	}
	tool, seen = importer()
	if code, _, _ := run(tool, "import-agent", "--from="); code != exitOK ||
		seen.from != "" || !seen.given {
		t.Errorf("an empty value: code %d, heard %+v", code, seen)
	}
}

// A lone -- ends the flags: what follows is words, however it looks.
func TestTwoHyphensEndTheFlags(t *testing.T) {
	tool, seen := importer()
	code, _, errs := run(tool, "import-agent", "a", "--", "--loud", "-f")
	if code != exitOK || seen.loud ||
		strings.Join(seen.args, " ") != "a --loud -f" {
		t.Errorf("code %d, errs %q, heard %+v", code, errs, seen)
	}
}

// A flag's value is what the parser takes for it, as cobra's pflag
// takes it: after =, or the next word, whatever it looks like, or the
// rest of a short flag's cluster; pzm does not second-guess it. Short
// switches cluster. doc/flags.md says each of these.
func TestAValueIsWhatTheParserTakes(t *testing.T) {
	for _, each := range []struct {
		asked, from string
		loud        bool
	}{
		{"import-agent --from=-x", "-x", false},
		{"import-agent --from -", "-", false},
		{"import-agent --from --loud", "--loud", false},
		{"import-agent -fada", "ada", false},
		{"import-agent -lf ada", "ada", true},
		{"import-agent -fl", "l", false},
	} {
		tool, seen := importer()
		code, _, errs := run(tool, strings.Fields(each.asked)...)
		if code != exitOK || seen.from != each.from || seen.loud != each.loud {
			t.Errorf("%s: code %d, errs %q, heard %+v", each.asked, code,
				errs, seen)
		}
	}
}

// A switch takes no value, "true" no more than any other, but its own
// long form counts as given: that is the text the parser hands a bare
// switch, and doc/flags.md says so. A list's empty value is an empty
// list, and an empty item in one is refused.
func TestASwitchAndAListTakeWhatTheyTake(t *testing.T) {
	for _, value := range []string{"true", "yes", "", "--loudly"} {
		tool, seen := importer()
		code, _, errs := run(tool, "import-agent", "--loud="+value)
		if code != exitRefused || !strings.Contains(errs, "a switch takes no value") ||
			seen.args != nil {
			t.Errorf("--loud=%s: code %d, errs %q", value, code, errs)
		}
	}
	for _, asked := range []string{"--loud=--loud", "-l=--loud"} {
		tool, seen := importer()
		if code, _, errs := run(tool, "import-agent",
			asked); code != exitOK || !seen.loud {
			t.Errorf("%s: code %d, errs %q, heard %+v", asked, code, errs,
				seen)
		}
	}
	tool, seen := importer()
	if code, _, _ := run(tool, "import-agent", "--tags="); code != exitOK ||
		seen.tags == nil || len(seen.tags) != none {
		t.Errorf("--tags=: code %d, tags %q", code, seen.tags)
	}
	for _, value := range []string{"a,,b", ",a", "a,"} {
		tool, seen := importer()
		code, _, errs := run(tool, "import-agent", "--tags", value)
		if code != exitRefused || !strings.Contains(errs, "has an empty item") ||
			seen.args != nil {
			t.Errorf("--tags %s: code %d, errs %q", value, code, errs)
		}
	}
}

// The tool's own flags are every verb's: before the verb, among its
// words, and before a verb asked for by its flag.
func TestTheToolsFlagsAreEveryVerbs(t *testing.T) {
	for _, asked := range [][]string{
		{"--roots", "r", "import-agent", "x"},
		{"import-agent", "x", "-r", "r"},
		{"--roots=r", "--import-agent", "x"},
		{"-r", "r", "-i", "x"},
	} {
		tool, seen := importer()
		code, _, errs := run(tool, asked...)
		if code != exitOK || seen.roots != "r" ||
			strings.Join(seen.args, " ") != "x" {
			t.Errorf("%v: code %d, errs %q, heard %+v", asked, code, errs,
				seen)
		}
	}
}

// The tool's own flags with no verb are refused: they are for a verb,
// and none ran.
func TestTheToolsFlagsWithNoVerb(t *testing.T) {
	tool, seen := importer()
	code, out, errs := run(tool, "--roots", "r")
	if code != exitRefused || out != "" || errs != "demo: --roots given, and no "+
		"verb: a verb comes after them\n" || seen.args != nil {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// A flag the parser will not take is refused in cobra's own words, with
// nothing added, and the tool exits 2 having run nothing.
func TestFlagErrorsAreCobras(t *testing.T) {
	for asked, says := range map[string]string{
		"import-agent --bogus":   "unknown flag: --bogus",
		"import-agent -q":        "unknown shorthand flag: 'q' in -q",
		"import-agent -lq":       "unknown shorthand flag: 'q' in -q",
		"import-agent --roots-x": "unknown flag: --roots-x",
		"--bogus plain":          "unknown flag: --bogus",
		"plain --from x":         "unknown flag: --from",
		"-r":                     "flag needs an argument: 'r' in -r",
		"--roots":                "flag needs an argument: --roots",
		"import-agent x --from":  "flag needs an argument: --from",
		"import-agent --loud=yes": "invalid argument \"yes\" for " +
			"\"-l, --loud\" flag: a switch takes no value",
		"import-agent -l --loud": "invalid argument \"--loud\" for " +
			"\"-l, --loud\" flag: it is given twice",
	} {
		tool, seen := importer()
		code, out, errs := run(tool, strings.Fields(asked)...)
		if code != exitRefused || out != "" || errs != "demo: "+says+"\n" ||
			seen.args != nil {
			t.Errorf("%s: code %d, out %q, errs %q, want %q", asked, code,
				out, errs, says)
		}
	}
}

// A flag given twice is refused: which was meant is not pzm's to guess.
func TestAFlagGivenTwiceIsRefused(t *testing.T) {
	for _, asked := range [][]string{
		{"import-agent", "--from", "a", "--from", "b"},
		{"import-agent", "-l", "--loud"},
		{"import-agent", "--tags", "a", "--tags", "b"},
		{"-r", "a", "import-agent", "--roots", "b"},
	} {
		tool, seen := importer()
		code, _, errs := run(tool, asked...)
		if code != exitRefused || !strings.Contains(errs, "it is given twice") ||
			seen.args != nil {
			t.Errorf("%v: code %d, errs %q", asked, code, errs)
		}
	}
}

// A flag not made by Switch, Value or List keeps the tool from running.
func TestAFlagPzmDidNotMakeIsRefused(t *testing.T) {
	tool, seen := importer()
	tool.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, func(pzm.Call) error {
		return nil
	}, pzm.Flag{})
	code, _, errs := run(tool, "v")
	if code != exitRefused || !strings.Contains(errs, "a flag not made by "+
		"pzm.Switch") || seen.args != nil || tool.Check() == nil {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// A verb's flag found after its words is refused: the verb comes first,
// so the flags after it are its own.
func TestAVerbAskedForLateIsRefused(t *testing.T) {
	tool, seen := built()
	code, _, errs := run(tool, "x", "--foo-bar")
	if code != exitRefused || !strings.Contains(errs, "--foo-bar comes first") ||
		len(seen.said) != none {
		t.Errorf("code %d, errs %q, ran %v", code, errs, seen.said)
	}
	tool, seen = built()
	code, _, errs = run(tool, "x", "--foo-bar", "--baz")
	if code != exitRefused || !strings.Contains(errs, "one verb at a time") ||
		len(seen.said) != none {
		t.Errorf("two late: code %d, errs %q, ran %v", code, errs,
			seen.said)
	}
}

// A verb or a flag named for what pzm gives every tool, or every verb of
// a kind, is refused, saying so and where to read about it.
func TestWhatPzmGivesIsNotDeclared(t *testing.T) {
	for says, make := range map[string]func(tool *pzm.Tool){
		"verb v: flag --dry-run: pzm gives --dry-run to every verb " +
			"whose facts say it writes; see doc/verbs.md": func(tool *pzm.Tool) {
			tool.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing,
				pzm.Switch("dry-run", 'n', "h"))
		},
		"tool demo: flag --share: pzm gives --share to every heavy " +
			"verb; see doc/verbs.md": func(tool *pzm.Tool) {
			tool.Flags(pzm.Value("share", verb.NoShort, "h"))
		},
		"verb check-config: pzm gives check-config to every tool; " +
			"see doc/verbs.md": func(tool *pzm.Tool) {
			tool.Verb("check-config", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing)
		},
		"verb edit-config: pzm gives edit-config": func(tool *pzm.Tool) {
			tool.Verb("edit-config", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing)
		},
		"verb seal: pzm gives seal": func(tool *pzm.Tool) {
			tool.Verb("seal", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing)
		},
	} {
		tool, seen := built()
		make(tool)
		code, _, errs := run(tool, "foo-bar")
		if code != exitRefused || !strings.Contains(errs, says) ||
			len(seen.said) != none || tool.Check() == nil {
			t.Errorf("code %d, errs %q, want %q", code, errs, says)
		}
	}
}

// declaredWrong is a declaration that keeps the tool from running, and
// the words its refusal holds.
type declaredWrong struct {
	why  string
	make func(tool *pzm.Tool)
	says string
}

// neverRuns holds that each declaration keeps the test tool from
// running, and that Check says why.
func neverRuns(t *testing.T, cases []declaredWrong) {
	t.Helper()
	for _, each := range cases {
		tool, seen := built()
		each.make(tool)
		code, _, errs := run(tool, "foo-bar")
		if code != exitRefused || !strings.Contains(errs, each.says) ||
			len(seen.said) != none {
			t.Errorf("%s: code %d, errs %q", each.why, code, errs)
		}
		if tool.Check() == nil {
			t.Errorf("%s: Check let it in", each.why)
		}
	}
}

// nothing is a verb that does nothing.
func nothing(pzm.Call) error { return nil }

// A verb's flags that cannot be declared keep the tool from running:
// one that cannot be a flag, pzm's own, and two sharing a name or a
// short flag.
func TestAVerbsFlagsDeclaredWrongNeverRun(t *testing.T) {
	verbWith := func(flags ...pzm.Flag) func(*pzm.Tool) {
		return func(tool *pzm.Tool) {
			tool.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing, flags...)
		}
	}
	neverRuns(t, []declaredWrong{
		{"a bad name", verbWith(pzm.Switch("Bad Name", verb.NoShort, "h")),
			"Bad Name"},
		{"pzm's name", verbWith(pzm.Switch("who", verb.NoShort, "h")), "pzm's own"},
		{"pzm's short", verbWith(pzm.Switch("hush", 'h', "h")),
			"pzm's own"},
		{"two names", verbWith(pzm.Switch("a", verb.NoShort, "h"),
			pzm.Value("a", verb.NoShort, "h")), "--a's"},
		{"two shorts", verbWith(pzm.Switch("a", 'x', "h"),
			pzm.Value("b", 'x', "h")), "--a's"},
	})
}

// The tool's own flags that cannot be declared keep it from running: one
// given twice, and one that is a verb's name, short flag, or one of its
// flags.
func TestTheToolsFlagsDeclaredWrongNeverRun(t *testing.T) {
	neverRuns(t, []declaredWrong{
		{"the tool's twice", func(tool *pzm.Tool) {
			tool.Flags(pzm.Switch("a", verb.NoShort, "h"), pzm.Switch("a", verb.NoShort, "h"))
		}, "tool demo"},
		{"a verb's name", func(tool *pzm.Tool) {
			tool.Flags(pzm.Switch("foo-bar", verb.NoShort, "h"))
		}, "verb foo-bar's"},
		{"a verb's short", func(tool *pzm.Tool) {
			tool.Flags(pzm.Switch("loud", 'f', "h"))
		}, "verb foo-bar's"},
		{"a verb's flag", func(tool *pzm.Tool) {
			tool.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, nothing,
				pzm.Value("from", 'q', "h"))
			tool.Flags(pzm.Switch("quiet", 'q', "h"))
		}, "verb v has its own --from"},
	})
}

// A verb that asks for a flag it did not declare, or as another kind, is
// a bug in the tool, and the panic says which.
func TestAskingForAnUndeclaredFlagPanics(t *testing.T) {
	for _, ask := range []func(pzm.Call){
		func(call pzm.Call) { call.On("nope") },
		func(call pzm.Call) { call.Value("nope") },
		func(call pzm.Call) { call.List("nope") },
		func(call pzm.Call) { call.Given("nope") },
		func(call pzm.Call) { call.Value("loud") },
	} {
		said := ""
		tool := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
		tool.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, func(call pzm.Call) error {
			defer func() { said, _ = recover().(string) }()
			ask(call)
			return nil
		}, pzm.Switch("loud", verb.NoShort, "h"))
		run(tool, "v")
		if !strings.Contains(said, "v asks for --") ||
			!strings.Contains(said, "which it does not declare") {
			t.Errorf("panicked with %q", said)
		}
	}
}

// RunWith hands the verb what it reads; Run hands it standard input.
func TestAVerbReadsWhatItIsGiven(t *testing.T) {
	tool, seen := importer()
	var out, errs bytes.Buffer
	code := tool.RunWith([]string{"import-agent"},
		strings.NewReader("a payload"), &out, &errs)
	if code != exitOK || seen.in != "a payload" {
		t.Errorf("code %d, errs %q, read %q", code, errs.String(), seen.in)
	}
	var stdin any
	plain := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	plain.Verb("v", verb.NoShort, "h", "p", "ExampleV", readsOnly, func(call pzm.Call) error {
		stdin = call.In()
		return nil
	})
	run(plain, "v")
	if stdin != os.Stdin {
		t.Errorf("Run gave the verb %v, not standard input", stdin)
	}
}

// A verb's help lists its flags and the tool's, the tool's help the
// tool's; for an agent, each flag is a line of its own.
func TestHelpSaysTheFlags(t *testing.T) {
	tool, _ := importer()
	_, out, _ := run(tool, "import-agent", "--help")
	want := "demo import-agent [ARGS]: imports an agent\n" +
		"the same as demo --import-agent or demo -i\n" +
		"  --from, -f VALUE    the old home\n" +
		"  --loud, -l          says more\n" +
		"  --tags A,B          words to keep\n" +
		"and for every verb:\n" +
		"  --roots, -r VALUE    where the roots are\n"
	if out != want {
		t.Errorf("verb help:\n%s", out)
	}
	tool, _ = importer()
	if _, out, _ := run(tool, "--help"); !strings.Contains(out,
		"for every verb:\n  --roots, -r VALUE    where the roots are\n") {
		t.Errorf("tool help:\n%s", out)
	}
	tool, _ = importer()
	_, out, _ = run(tool, "--agent-help")
	for _, line := range []string{
		"flag --roots, -r VALUE: where the roots are\n",
		"flag import-agent --from, -f VALUE: the old home\n",
		"flag import-agent --tags A,B: words to keep\n",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("agent help has no %q:\n%s", line, out)
		}
	}
}

// Anti-test: after a lone --, a word that starts with a hyphen is a word
// for the verb, and the -- itself is not passed on.
func TestAHyphenWordAfterTwoHyphensIsAWord(t *testing.T) {
	tool, seen := importer()
	code, _, errs := run(tool, "import-agent", "--", "-fast")
	if code != exitOK || strings.Join(seen.args, " ") != "-fast" {
		t.Errorf("code %d, errs %q, heard %+v", code, errs, seen)
	}
}
