package pzm

import (
	"fmt"
	"time"

	"github.com/janearc/puzzlemouth/internal/age"
	"github.com/janearc/puzzlemouth/internal/who"
	"github.com/janearc/puzzlemouth/verb"
)

// Build is the short commit game build stamps into main.build.
type Build string

// Built is that commit's time, which game build stamps into main.built.
type Built string

// Who is the session and machine that ran the build, from main.who.
type Who string

// Sessions are the last sessions to commit behind the build. main.sids
// holds them joined with commas.
type Sessions string

// stamps are what game built into a tool; a tool built by go has none.
type stamps struct {
	build Build
	built Built
	who   Who
	sids  Sessions
}

// builtinVerbs are the verbs every tool has from pzm: --age and --who.
func (tool *Tool) builtinVerbs() []declaredVerb {
	return []declaredVerb{
		tool.builtinVerb("age", "which commit this is, and how long "+
			"ago it was made", tool.age),
		tool.builtinVerb("who",
			"who built this, and the sessions behind it", tool.who),
	}
}

// builtinVerb is one of pzm's own verbs, shown in pzm's own example.
func (tool *Tool) builtinVerb(name verb.Name, help verb.Help,
	do Do) declaredVerb {
	declared, _ := verb.New(name, verb.NoShort, help,
		"# "+verb.Page(name), "ExampleNew",
		verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
			verb.ThisMachine, verb.NoOne, verb.Light))
	return declaredVerb{verb: *declared, do: do}
}

// age is --age: the tool's commit and how long ago it was made.
func (tool *Tool) age(call Call) error {
	stamps := tool.stamps
	said := age.Of(string(tool.name), string(stamps.build),
		string(stamps.built), time.Now())
	_, err := fmt.Fprintln(call.Out(), said)
	return err
}

// who is --who: who built the tool and the sessions behind it. It also
// says whether the signature beside it checks.
func (tool *Tool) who(call Call) error {
	vouch, err := tool.signedBy()
	if err != nil {
		return err
	}
	said := who.Of(string(tool.name), string(tool.stamps.who),
		string(tool.stamps.sids), vouch.signer)
	if vouch.unset {
		said += "\nno signer is set: no key is vouched for; add a " +
			"signer line to " + vouch.path
	}
	_, err = fmt.Fprintln(call.Out(), said)
	return err
}

// vouch is who signed the running binary, as far as the config vouches.
type vouch struct {
	signer string // the name of the signer, or nobody
	unset  bool   // no signer is set in the config at path
	path   string // the config's path
}

// signedBy is who signed the running binary, by the signature beside it,
// among the signers the config sets. It is nobody when there is no
// signature, and an error when one does not check or is by a stranger.
//
// When no signer is set it is nobody and unset says why.
func (tool *Tool) signedBy() (*vouch, error) {
	set, err := tool.settings()
	if err != nil {
		return nil, err
	}
	if !set.signers.Any() {
		return &vouch{unset: true, path: set.path}, nil
	}
	signer, err := who.SignedBy(tool.executable, set.signers)
	if err != nil {
		return nil, err
	}
	return &vouch{signer: signer, path: set.path}, nil
}
