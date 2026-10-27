package pzm_test

import (
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
	"pgregory.net/rapid"
)

// twice is a tool whose flags were declared in two calls.
func twice(first, second pzm.Flag) *pzm.Tool {
	tool, _ := built()
	tool.Flags(first)
	tool.Flags(second)
	return tool
}

// Flags called twice with one name is refused by Check and by a run, in
// the tool's words. It never reaches the parser, which would panic.
func TestFlagsDeclaredTwiceByNameAreRefused(t *testing.T) {
	tool := twice(pzm.Switch("quiet", 'q', "say less"),
		pzm.Value("quiet", 'z', "say nothing"))
	if err := tool.Check(); err == nil {
		t.Errorf("Check passed two tool flags named quiet")
	}
	for _, args := range [][]string{{}, {"--help"}, {"foo-bar"}} {
		code, out, errs := run(tool, args...)
		if code != refusedCode || out != "" ||
			!strings.Contains(errs, "--quiet") {
			t.Errorf("%v: code %d, out %q, errs %q", args, code, out, errs)
		}
	}
}

// The same for a short flag two calls share.
func TestFlagsDeclaredTwiceByShortAreRefused(t *testing.T) {
	tool := twice(pzm.Switch("quiet", 'q', "say less"),
		pzm.Value("queue", 'q', "which queue"))
	if err := tool.Check(); err == nil {
		t.Errorf("Check passed two tool flags with short -q")
	}
	code, _, errs := run(tool, "--help")
	if code != refusedCode || !strings.Contains(errs, "--queue") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// Anti-test: two calls that declare different flags are not refused.
func TestFlagsDeclaredInTwoCallsAreKept(t *testing.T) {
	tool := twice(pzm.Switch("quiet", 'q', "say less"),
		pzm.Value("queue", 'Q', "which queue"))
	if err := tool.Check(); err != nil {
		t.Errorf("Check refused distinct flags: %v", err)
	}
	code, _, errs := run(tool, "-q", "-Q", "x", "foo-bar")
	if code != ranCode {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// doNothing is a verb that runs and succeeds.
func doNothing(pzm.Call) error { return nil }

// The caller owns the flag slice given to Verb. Clearing it after the
// verb is declared changes nothing the tool checked or runs.
func TestAVerbsFlagsAreCopied(t *testing.T) {
	tool, _ := built()
	flags := []pzm.Flag{pzm.Switch("loud", 'l', "say it louder")}
	tool.Verb("shout", 's', "shouts", "# shout", "ExampleShout", readsOnly,
		doNothing, flags...)
	clear(flags)
	if err := tool.Check(); err != nil {
		t.Fatalf("Check: %v", err)
	}
	code, _, errs := run(tool, "shout", "--loud")
	if code != ranCode {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// The caller owns the flag slice given to Flags. Clearing it after the
// flags are declared changes nothing the tool runs.
func TestTheToolsFlagsAreCopied(t *testing.T) {
	tool, _ := built()
	flags := []pzm.Flag{pzm.Switch("quiet", 'q', "say less")}
	tool.Flags(flags...)
	clear(flags)
	code, _, errs := run(tool, "-q", "foo-bar")
	if code != ranCode {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

const (
	// firstCall is the first call to Flags the property below spreads
	// its flags over. It's defined.
	firstCall = 0
	// lastCall is the last of them. It's defined.
	lastCall = 2
	// fewest is the fewest flags it draws. It's defined.
	fewest = 1
	// most is the most flags it draws. It's defined.
	most = 5
)

// flagNames and flagShorts are what a flag in the property below may be
// called. Few, so that clashes happen often.
var (
	flagNames  = []verb.Name{"one", "two", "three"}
	flagShorts = []verb.Short{verb.NoShort, 'x', 'y', 'z'}
)

// drawnFlag is one flag the property drew, and the call that declares it.
type drawnFlag struct {
	name  verb.Name
	short verb.Short
	call  int
}

// drawFlag draws one flag and the call that declares it.
func drawFlag(t *rapid.T) drawnFlag {
	return drawnFlag{rapid.SampledFrom(flagNames).Draw(t, "name"),
		rapid.SampledFrom(flagShorts).Draw(t, "short"),
		rapid.IntRange(firstCall, lastCall).Draw(t, "call")}
}

// share is whether two flags share a name or a short flag.
func share(one, other drawnFlag) bool {
	return one.name == other.name ||
		one.short != verb.NoShort && one.short == other.short
}

// clashing is whether any two of the drawn flags share a name or a short
// flag.
func clashing(drawn []drawnFlag) bool {
	for i, one := range drawn {
		for _, other := range drawn[:i] {
			if share(one, other) {
				return true
			}
		}
	}
	return false
}

// declareOver declares the drawn flags to the tool, each call's flags in
// one call to Flags.
func declareOver(tool *pzm.Tool, drawn []drawnFlag) {
	for call := firstCall; call <= lastCall; call++ {
		var batch []pzm.Flag
		for _, each := range drawn {
			if each.call == call {
				batch = append(batch, pzm.Switch(each.name, each.short,
					"a switch"))
			}
		}
		tool.Flags(batch...)
	}
}

// Flags declared over several calls are refused when two share a name or
// a short flag, and only then. Help never panics. Refused, it exits 2.
// Sound, it exits 0.
func TestFlagsOverManyCallsAreRefusedWhenTheyClash(t *testing.T) {
	rapid.Check(t, checkManyCalls)
}

// checkManyCalls is one draw of that property.
func checkManyCalls(t *rapid.T) {
	drawn := rapid.SliceOfN(rapid.Custom(drawFlag), fewest,
		most).Draw(t, "flags")
	tool, _ := built()
	declareOver(tool, drawn)
	code, _, errs := run(tool, "--help")
	refused := tool.Check() != nil
	if clashing(drawn) != (code == refusedCode) || refused != (code == refusedCode) {
		t.Fatalf("clash %v, code %d, refused %v, errs %q",
			clashing(drawn), code, refused, errs)
	}
}
