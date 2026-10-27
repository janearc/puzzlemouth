package pzm_test

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// given is the tool's own flags as one call spells them. It holds the
// words and what they say: whether the switch is on, and the roots.
type given struct {
	words []string
	quiet bool
	roots string
}

// rootsValues are words a roots value may be. Each names a verb or asks
// for one. One is a plain word. None is the verb that runs.
var rootsValues = []string{"tidy", "look", "plain", "boom", "age", "who",
	"help", "--look", "-l", "--tidy", "-t", "--age", "elsewhere"}

// quietWords are the ways to give the switch.
var quietWords = []string{"-q", "--quiet"}

// rootsWords are the ways to give the roots value, with value as it.
func rootsWords(value string) [][]string {
	return [][]string{{"-r", value}, {"-r" + value}, {"-r=" + value},
		{"--roots", value}, {"--roots=" + value}}
}

// spellings are every way to give the tool's own flags in one call.
// The switch comes before the value when both are given, so they can
// cluster as -qr.
func spellings(value string) []given {
	var all []given
	all = append(all, given{words: []string{}})
	for _, quiet := range quietWords {
		all = append(all, given{words: []string{quiet}, quiet: true})
	}
	for _, roots := range rootsWords(value) {
		all = append(all, given{words: roots, roots: value})
		for _, quiet := range quietWords {
			before := append([]string{quiet}, roots...)
			after := append(append([]string{}, roots...), quiet)
			all = append(all, given{before, true, value},
				given{after, true, value})
		}
	}
	all = append(all, given{[]string{"-qr", value}, true, value},
		given{[]string{"-qr" + value}, true, value},
		given{[]string{"-qr=" + value}, true, value})
	return all
}

// genGiven draws one spelling of the tool's own flags.
func genGiven(t *rapid.T) given {
	value := rapid.SampledFrom(rootsValues).Draw(t, "value")
	return rapid.SampledFrom(spellings(value)).Draw(t, "spelling")
}

// verbsRun are the verbs a call may run, by name.
var verbsRun = []string{"tidy", "look", "plain"}

// The verb after the tool's own flags is the one that runs. That holds
// however they're spelled and whatever their value is. The flags hold
// what was given. Even a verb's name is only a value there.
func TestTheVerbAfterTheToolsFlagsIsTheOneThatRuns(t *testing.T) {
	rapid.Check(t, checkTheVerbRuns)
}

// checkTheVerbRuns is one draw of that property.
func checkTheVerbRuns(t *rapid.T) {
	lead := genGiven(t)
	name := rapid.SampledFrom(verbsRun).Draw(t, "verb")
	args := append(append([]string{}, lead.words...), name, "word")
	var out, errs bytes.Buffer
	code := dispatchTool().Run(args, &out, &errs)
	want := fmt.Sprintf("ran %s words=[\"word\"] quiet=%t roots=%q",
		name, lead.quiet, lead.roots)
	if code != ranCode || !strings.HasPrefix(out.String(), want) {
		t.Fatalf("%q: code %d, out %q, errs %q; want %q", args, code,
			out.String(), errs.String(), want)
	}
}

// anti-tests: the word after the tool's own flags is refused when it's
// no verb. That holds however the flags are spelled. No verb named by
// their values runs in its place.
func TestAWordThatIsNoVerbRunsNoVerb(t *testing.T) {
	rapid.Check(t, checkNoVerbRuns)
}

// checkNoVerbRuns is one draw of that property.
func checkNoVerbRuns(t *rapid.T) {
	lead := genGiven(t)
	args := append(append([]string{}, lead.words...), "nonsuch", "tidy")
	var out, errs bytes.Buffer
	code := dispatchTool().Run(args, &out, &errs)
	if code != refusedCode || out.String() != "" ||
		errs.String() != "demo: there is no verb nonsuch\n" {
		t.Fatalf("%q: code %d, out %q, errs %q", args, code,
			out.String(), errs.String())
	}
}

const (
	// ranCode is the exit code of a call whose verb ran. It is defined by
	// pzm.Tool.RunWith: 0.
	ranCode = 0
	// refusedCode is the exit code of a call that ran nothing. It is
	// defined by pzm.Tool.RunWith: 2.
	refusedCode = 2
)

// outcome is what the tool does to args: its exit code and what each
// writer got.
func outcome(args []string) (int, string, string) {
	var out, errs bytes.Buffer
	code := dispatchTool().Run(args, &out, &errs)
	return code, out.String(), errs.String()
}

// verbFlags are the two flags that ask for each verb a call may run.
var verbFlags = map[string][]string{"tidy": {"--tidy", "-t"},
	"look": {"--look", "-l"}, "plain": {"--plain", "-p"}}

// Asking for a verb by its flag does what asking by its name does. The
// flag is long or short. That holds however the tool's own flags before
// it are spelled.
func TestAVerbAskedForByItsFlagRunsAsByItsName(t *testing.T) {
	rapid.Check(t, checkFlagRunsAsName)
}

// checkFlagRunsAsName is one draw of that property.
func checkFlagRunsAsName(t *rapid.T) {
	lead := genGiven(t)
	name := rapid.SampledFrom(verbsRun).Draw(t, "verb")
	asked := rapid.SampledFrom(verbFlags[name]).Draw(t, "flag")
	byName := append(slices.Clone(lead.words), name, "word")
	byFlag := append(slices.Clone(lead.words), asked, "word")
	nameCode, nameOut, nameErrs := outcome(byName)
	flagCode, flagOut, flagErrs := outcome(byFlag)
	if nameCode != flagCode || nameOut != flagOut || nameErrs != flagErrs {
		t.Fatalf("%q gave %d %q %q; %q gave %d %q %q", byName, nameCode,
			nameOut, nameErrs, byFlag, flagCode, flagOut, flagErrs)
	}
}

// stray is a flag that cannot be read after the tool's own flags. It
// holds what the tool says to it after "demo: ". The word after it is
// no verb.
type stray struct{ flag, says string }

// noVerb is what the tool says to a verb's flag with no verb after it.
// The word after it is no verb.
const noVerb = " is a flag of a verb, and word is no verb: a verb's " +
	"flags come after it"

// strayFlags are flags that are neither the tool's own nor a verb's
// name. Some are verbs' own flags. Some are flags nothing has. Some run
// verbs' flags together, or with the tool's.
var strayFlags = []stray{{"--force", "--force" + noVerb},
	{"-f", "-f" + noVerb}, {"--from=x", "--from" + noVerb},
	{"-F", "-F is a flag of a verb, and no verb comes after it"},
	{"--dry-run", "--dry-run" + noVerb},
	{"-z", "unknown shorthand flag: 'z' in -z"},
	{"--bogus", "unknown flag: --bogus"},
	{"-qz", "unknown shorthand flag: 'z' in -z"},
	{"-lt", "one verb at a time"}}

// anti-tests: a value given to the tool's own flags is never run as the
// verb. Here the tool's flags are followed by a stray flag. So no verb
// runs at all, even when one of the tool's values names one. The refusal
// is the stray flag's, and no other.
func TestAToolFlagsValueIsNeverRunAsTheVerb(t *testing.T) {
	rapid.Check(t, checkValueNeverRuns)
}

// checkValueNeverRuns is one draw of that property.
func checkValueNeverRuns(t *rapid.T) {
	lead := genGiven(t)
	each := rapid.SampledFrom(strayFlags).Draw(t, "stray")
	args := append(slices.Clone(lead.words), each.flag, "word")
	code, out, errs := outcome(args)
	if code != refusedCode || out != "" ||
		errs != "demo: "+each.says+"\n" {
		t.Fatalf("%q: code %d, out %q, errs %q; want %q", args, code, out,
			errs, each.says)
	}
}

// nextWords are words a call may have after the tool's own flags.
var nextWords = []string{"tidy", "--look", "-l", "nonsuch", "--force",
	"-z", "--", "--help"}

// anti-tests: running a call never changes the words it was given.
func TestTheCallersWordsAreNeverChanged(t *testing.T) {
	rapid.Check(t, checkWordsKept)
}

// checkWordsKept is one draw of that property.
func checkWordsKept(t *rapid.T) {
	lead := genGiven(t)
	next := rapid.SampledFrom(nextWords).Draw(t, "next")
	args := append(slices.Clone(lead.words), next, "word")
	kept := slices.Clone(args)
	outcome(args)
	if !slices.Equal(args, kept) {
		t.Fatalf("the call changed its words from %q to %q", kept, args)
	}
}
