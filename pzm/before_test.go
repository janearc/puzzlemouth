package pzm_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"pgregory.net/rapid"
)

// mixFlag is a flag of a verb in mixTool: its name, and whether it takes
// a value.
type mixFlag struct {
	name  string
	takes bool
}

// mixFlags are the flags each verb of mixTool declares, in the order its
// line says them. Same and via are declared by two verbs and mean the
// same to both. Odd is a switch to tidy and takes a value for look, so
// it cannot be read before the verb.
var mixFlags = map[string][]mixFlag{
	"tidy":  {{"force", false}, {"same", false}, {"via", true}, {"odd", false}},
	"look":  {{"from", true}, {"same", false}, {"via", true}, {"odd", true}},
	"plain": nil,
}

// mixing is a verb that writes down that it ran. It gives its name and
// its words. Then it gives every flag it declares with what it was given.
type mixing struct{ name string }

// do is the verb's run. It writes one line.
func (say mixing) do(call pzm.Call) error {
	var line strings.Builder
	fmt.Fprintf(&line, "ran %s words=%q quiet=%t roots=%q", say.name,
		call.Args(), call.On("quiet"), call.Value("roots"))
	for _, each := range mixFlags[say.name] {
		if each.takes {
			fmt.Fprintf(&line, " %s=%q", each.name, call.Value(each.name))
			continue
		}
		fmt.Fprintf(&line, " %s=%t", each.name, call.On(each.name))
	}
	_, err := fmt.Fprintln(call.Out(), line.String())
	return err
}

// mixTool has a switch and a value of its own. Its verbs are tidy, look
// and plain. Their flags are in mixFlags.
func mixTool() *pzm.Tool {
	tool := pzm.New("mix", "a tool for testing flags before the verb", "",
		"", "", "")
	tool.Flags(pzm.Switch("quiet", 'q', "says less"),
		pzm.Value("roots", 'r', "where the roots are"))
	tool.Verb("tidy", 't', "takes away what is old", "# tidy",
		"ExampleTidy", readsOnly, mixing{"tidy"}.do,
		pzm.Switch("force", 'f', "does not ask"),
		pzm.Switch("same", 's', "is the same switch to two verbs"),
		pzm.Value("via", 'v', "is the same value to two verbs"),
		pzm.Switch("odd", 'x', "is a switch here and a value there"))
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", readsOnly,
		mixing{"look"}.do,
		pzm.Value("from", 'F', "where from"),
		pzm.Switch("same", 's', "is the same switch to two verbs"),
		pzm.Value("via", 'v', "is the same value to two verbs"),
		pzm.Value("odd", 'x', "is a switch there and a value here"))
	tool.Verb("plain", 'p', "takes no flags", "# plain", "ExamplePlain",
		readsOnly, mixing{"plain"}.do)
	return tool
}

// mixed is what mixTool says to args: its exit code and what each writer
// got.
func mixed(args []string) (int, string, string) {
	return run(mixTool(), args...)
}

// line is what a verb of mixTool writes when it ran. It ran with words
// and the tool's flags. Given holds the verb's own flags and what each
// was given. Any text but "" means a switch was given.
func line(name string, words []string, quiet bool, roots string,
	given map[string]string) string {
	var said strings.Builder
	fmt.Fprintf(&said, "ran %s words=%q quiet=%t roots=%q", name, words,
		quiet, roots)
	for _, each := range mixFlags[name] {
		switch {
		case each.takes:
			fmt.Fprintf(&said, " %s=%q", each.name, given[each.name])
		default:
			fmt.Fprintf(&said, " %s=%t", each.name,
				given[each.name] != "")
		}
	}
	return said.String() + "\n"
}

// Two verbs may declare a flag alike. It is read before either verb. It
// goes to the verb that runs.
func TestAFlagTwoVerbsShareIsReadBeforeTheVerb(t *testing.T) {
	on := map[string]string{"same": "on"}
	via := map[string]string{"via": "cy"}
	for _, each := range []struct {
		args []string
		want string
	}{
		{[]string{"-s", "tidy"}, line("tidy", []string{}, false, "", on)},
		{[]string{"-s", "look"}, line("look", []string{}, false, "", on)},
		{[]string{"--same", "-l"}, line("look", []string{}, false, "", on)},
		{[]string{"-qs", "tidy", "w"}, line("tidy", []string{"w"}, true,
			"", on)},
		{[]string{"-v", "cy", "tidy"}, line("tidy", []string{}, false, "",
			via)},
		{[]string{"-vcy", "look"}, line("look", []string{}, false, "",
			via)},
		{[]string{"--via=cy", "look"}, line("look", []string{}, false, "",
			via)},
		{[]string{"-v", "look", "look"}, line("look", []string{}, false,
			"", map[string]string{"via": "look"})},
	} {
		code, out, errs := mixed(each.args)
		if code != ranCode || out != each.want {
			t.Errorf("%q: code %d, out %q, errs %q; want %q", each.args,
				code, out, errs, each.want)
		}
	}
}

// Anti-tests: a verb's flag before the verb is refused. That holds when
// the verb after it does not declare the flag. It holds whatever the
// value is. It holds when the verbs disagree on the flag. Nothing runs.
func TestAVerbsFlagBeforeAnotherVerbIsRefused(t *testing.T) {
	for _, each := range []struct{ args, says string }{
		{"-s plain", "-s is tidy's and look's, not plain's"},
		{"-f look", "-f is tidy's, not look's"},
		{"-F tidy tidy", "-F is look's, not tidy's"},
		{"-v tidy plain", "-v is tidy's and look's, not plain's"},
		{"-x tidy", "-x is not the same flag to every verb: put it " +
			"after the verb"},
		{"--odd look", "--odd is not the same flag to every verb: put " +
			"it after the verb"},
		{"-sx tidy", "-x is not the same flag to every verb: put it " +
			"after the verb"},
		{"-xs plain", "-x is not the same flag to every verb: put it " +
			"after the verb"},
		{"-v", "-v is a flag of a verb, and no verb comes after it"},
		{"--via", "--via is a flag of a verb, and no verb comes after it"},
		{"-f --", "-f is a flag of a verb, and -- comes before any verb"},
		{"-f --help tidy", "-f is a flag of a verb, and --help comes " +
			"before any verb"},
		{"-f nonsuch", "-f is a flag of a verb, and nonsuch is no verb"},
		{"-f help tidy", "-f is a flag of a verb, and help is no verb"},
		{"-lt", "one verb at a time"},
		{"-tl", "one verb at a time"},
	} {
		code, out, errs := mixed(strings.Fields(each.args))
		if code != refusedCode || out != "" ||
			!strings.Contains(errs, "mix: "+each.says) {
			t.Errorf("%s: code %d, out %q, errs %q; want %q", each.args,
				code, out, errs, each.says)
		}
	}
}

// pick is a flag a call may give before the verb. It has a name and a
// letter. It takes a value or not. Owners are the verbs that declare it.
// The tool's own flags have none.
type pick struct {
	name   string
	letter string
	takes  bool
	owners []string
}

// picks are the flags a call draws from.
var picks = []pick{
	{"quiet", "q", false, nil},
	{"roots", "r", true, nil},
	{"force", "f", false, []string{"tidy"}},
	{"from", "F", true, []string{"look"}},
	{"same", "s", false, []string{"tidy", "look"}},
	{"via", "v", true, []string{"tidy", "look"}},
	{"odd", "x", false, []string{"tidy", "look"}},
}

// valueWords are words a flag's value may be. Each names a verb or asks
// for one. Some ask for help. Some are flags. None is the verb that runs.
var valueWords = []string{"tidy", "look", "plain", "help", "--look",
	"-l", "--tidy", "-t", "--plain", "-p", "--help", "-h", "--", "-f",
	"--force", "elsewhere"}

// shortOf is each verb's short flag, which asks for it.
var shortOf = map[string]string{"tidy": "t", "look": "l", "plain": "p"}

// mixCall is a call drawn for mixTool. It holds the words and the verb
// that is to run. It holds the flags given before the verb, by name.
type mixCall struct {
	args   []string
	verb   string
	given  map[string]string
	picks  []pick
	dashed bool // a lone dash stands before the word
}

// wordsFor is the words the verb is to be given.
func (call mixCall) wordsFor() []string {
	if call.dashed {
		return []string{"-", "word"}
	}
	return []string{"word"}
}

// valueForms are the ways to give a value flag its value, with value as
// the word. They hold the flag's letter and name.
func valueForms(each pick, value string) [][]string {
	return [][]string{{"-" + each.letter, value},
		{"-" + each.letter + value},
		{"-" + each.letter + "=" + value},
		{"--" + each.name, value}, {"--" + each.name + "=" + value}}
}

// genMix draws a call. Its flags are clustered or apart. Value flags may
// end a cluster. They may take the word after it. The verb is given by
// its name or its flag. It may also be given by its letter in the
// cluster. Values are drawn from words that name verbs. When own is set,
// every flag drawn is the tool's or the verb's. None is one that two
// verbs declare differently.
func genMix(t *rapid.T, own bool) mixCall {
	call := mixCall{given: map[string]string{}}
	call.verb = rapid.SampledFrom(verbsRun).Draw(t, "verb")
	pool := poolFor(call.verb, own)
	call.picks = rapid.SliceOfNDistinct(rapid.SampledFrom(pool), noFlags,
		len(pool), nameOf).Draw(t, "flags")
	words := &drawn{}
	clustered := rapid.Bool().Draw(t, "clustered")
	for at := range call.picks {
		each := &call.picks[at]
		call.given[each.name] = valueFor(t, *each)
		words.give(t, each, call.given[each.name], clustered)
	}
	call.args, call.dashed = words.finish(t, call.verb, call.given)
	return call
}

// noFlags is no flags, no letters and the first place. It's defined.
const noFlags = 0

// nameOf is the name of a flag, which tells two flags apart.
func nameOf(each pick) string { return each.name }

// outside is whether the flag is another verb's: it has verbs, and the
// verb is not among them.
func (each pick) outside(verb string) bool {
	return each.owners != nil && !slices.Contains(each.owners, verb)
}

// disputed is whether the verbs declare the flag differently.
func (each pick) disputed() bool { return each.name == "odd" }

// poolFor is the flags a call to the verb may draw. It is all of them.
// When own is set it is the tool's and the verb's. None is disputed.
func poolFor(verb string, own bool) []pick {
	if !own {
		return slices.Clone(picks)
	}
	var pool []pick
	for _, each := range picks {
		if !each.disputed() && !each.outside(verb) {
			pool = append(pool, each)
		}
	}
	return pool
}

// valueFor is what a call gives the flag: "on" for a switch, or a word
// drawn from valueWords.
func valueFor(t *rapid.T, each pick) string {
	if !each.takes {
		return "on"
	}
	return rapid.SampledFrom(valueWords).Draw(t, each.name)
}

// drawn is the words of a call as its flags are drawn into them.
type drawn struct {
	lead    []string // flags given apart, in the order drawn
	letters []string // switches clustered
	ending  *pick    // the value flag that ends the cluster
}

// give draws how one flag is given: apart, or in the cluster.
func (words *drawn) give(t *rapid.T, each *pick, value string,
	clustered bool) {
	switch {
	case each.takes && clustered && words.ending == nil:
		words.ending = each
	case each.takes:
		forms := rapid.SampledFrom(valueForms(*each, value))
		words.lead = append(words.lead, forms.Draw(t, "form")...)
	case clustered && !each.disputed():
		words.letters = append(words.letters, each.letter)
	default:
		spellings := []string{"-" + each.letter, "--" + each.name}
		spelled := rapid.SampledFrom(spellings).Draw(t, "spelling")
		words.lead = append(words.lead, spelled)
	}
}

// finish is the words of the call. They are the flags given apart and
// the cluster. The verb follows unless the cluster asks for it. One word
// for the verb ends them. The bool says whether a lone dash was drawn.
// It may stand between any two of those parts. Wherever it stands, it is
// a word for the verb.
func (words *drawn) finish(t *rapid.T, verb string,
	given map[string]string) ([]string, bool) {
	asks := rapid.Bool().Draw(t, "asks")
	if asks {
		at := rapid.IntRange(noFlags, len(words.letters)).Draw(t, "at")
		words.letters = slices.Insert(words.letters, at, shortOf[verb])
	}
	cluster := clusterOf(t, words.letters, words.ending, given)
	parts := [][]string{words.lead, cluster}
	if !asks {
		spellings := []string{verb, "--" + verb, "-" + shortOf[verb]}
		named := rapid.SampledFrom(spellings).Draw(t, "verb word")
		parts = append(parts, []string{named})
	}
	dashed := rapid.Bool().Draw(t, "dashed")
	if dashed {
		at := rapid.IntRange(noFlags, len(parts)).Draw(t, "dash at")
		parts = slices.Insert(parts, at, []string{"-"})
	}
	return append(slices.Concat(parts...), "word"), dashed
}

// clusterOf is the words of one cluster of short flags. It holds the
// letters. Value flags may end it. Their value is then attached or
// attached after an =. Or the value is the word after. It is no words
// when there are no letters and no such flag.
func clusterOf(t *rapid.T, letters []string, ending *pick,
	given map[string]string) []string {
	word := "-" + strings.Join(letters, "")
	if ending == nil && len(letters) == noFlags {
		return nil
	}
	if ending == nil {
		return []string{word}
	}
	value := given[ending.name]
	forms := [][]string{{word + ending.letter + value},
		{word + ending.letter + "=" + value}, {word + ending.letter, value}}
	return rapid.SampledFrom(forms).Draw(t, "ending")
}

// want is what the call is to do. The verb it names runs with the flags
// it was given. Or nothing runs and the refusal says why. Either the
// verbs disagree on a flag or a flag belongs to another verb.
func (call mixCall) want() (string, string) {
	var disagree, foreign bool
	for _, each := range call.picks {
		disagree = disagree || each.disputed()
		foreign = foreign || !each.disputed() && each.outside(call.verb)
	}
	switch {
	case disagree:
		return "", "is not the same flag to every verb: put it after " +
			"the verb"
	case foreign:
		return "", ", not " + call.verb + "'s"
	}
	return line(call.verb, call.wordsFor(), call.given["quiet"] != "",
		call.given["roots"], call.given), ""
}

// checkMix is one draw of the properties below.
func checkMix(t *rapid.T, own bool) {
	call := genMix(t, own)
	ran, says := call.want()
	if own && ran == "" {
		t.Fatalf("%q: flags of its own were refused", call.args)
	}
	code, out, errs := mixed(call.args)
	if strings.HasPrefix(out, "ran ") &&
		!strings.HasPrefix(out, "ran "+call.verb+" ") {
		t.Fatalf("%q ran a verb other than %s: %q", call.args, call.verb,
			out)
	}
	if ran != "" && (code != ranCode || out != ran) {
		t.Fatalf("%q: code %d, out %q, errs %q; want %q", call.args, code,
			out, errs, ran)
	}
	if ran == "" && (code != refusedCode || out != "" ||
		!strings.HasPrefix(errs, "mix: ") ||
		!strings.Contains(errs, says)) {
		t.Fatalf("%q: code %d, out %q, errs %q; want a refusal with %q",
			call.args, code, out, errs, says)
	}
}

// The tool's flags and verbs' flags come before the verb. They run
// together or apart. Their values are drawn from the names of verbs.
// Only the verb after them ever runs. Another verb's flag refuses the
// call. So does a flag the verbs disagree on. No verb named by a value
// is run.
func TestAVerbNamedByAValueNeverRuns(t *testing.T) {
	rapid.Check(t, checkAnyFlags)
}

// checkAnyFlags is one draw of that property.
func checkAnyFlags(t *rapid.T) { checkMix(t, false) }

// Anti-test of the one above. Every flag is the tool's or the verb's.
// Then the verb after them runs with every value as given.
func TestTheVerbRunsWithTheFlagsThatAreItsOwn(t *testing.T) {
	rapid.Check(t, checkOwnFlags)
}

// checkOwnFlags is one draw of that property.
func checkOwnFlags(t *rapid.T) { checkMix(t, true) }

// Anti-tests: after the verb a flag two verbs declare differently is that
// verb's, and the verb runs. Only before it can the flag not be read.
func TestAFlagTheVerbsDisagreeOnIsTheVerbsAfterIt(t *testing.T) {
	for args, want := range map[string]string{
		"tidy -x": line("tidy", []string{}, false, "",
			map[string]string{"odd": "on"}),
		"look -x v": line("look", []string{}, false, "",
			map[string]string{"odd": "v"}),
		"look --odd=v": line("look", []string{}, false, "",
			map[string]string{"odd": "v"}),
	} {
		code, out, errs := mixed(strings.Fields(args))
		if code != ranCode || out != want {
			t.Errorf("%s: code %d, out %q, errs %q; want %q", args, code,
				out, errs, want)
		}
	}
}
