package pzm

import (
	"slices"

	"github.com/janearc/puzzlemouth/verb"
)

// dryRun is the --dry-run pzm gives every verb whose facts say it writes.
// No verb declares its own. pzm hands the verb the switch and cannot see
// what it writes. So the line asks the verb and promises nothing for it.
// Keeping to it is the verb's; its example running dry is the one check.
var dryRun = newFlag("dry-run", verb.NoShort,
	"asks the verb to say what it would do, and do none of it",
	switchKind)

// allFlags are the verb's own flags and those pzm gives it: --dry-run,
// when its facts say it writes.
func (each declaredVerb) allFlags() []Flag {
	if !each.verb.Facts().WritesAnything() {
		return each.flags
	}
	return append(slices.Clone(each.flags), dryRun)
}

// DryRun is whether the verb was asked to say what it would do and do
// none of it. It is always false for a verb that writes nothing.
func (call Call) DryRun() bool {
	value, ok := call.flags["dry-run"]
	return ok && value.given
}
