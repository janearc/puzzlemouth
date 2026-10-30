package pzm

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/janearc/puzzlemouth/config"
	"github.com/janearc/puzzlemouth/internal/host"
	"github.com/janearc/puzzlemouth/internal/who"
	"github.com/janearc/puzzlemouth/verb"
)

// The keys of the lines pzm reads from its own config, ~/.config/pzm/
// config. They are lowercase words joined by hyphens, as config wants.
// firstWord is the place of the one word a line of the config takes. It's
// defined by declared: each setting takes one.
const firstWord = none

const (
	// signerKey is "signer NAME :: ssh-ed25519 KEY": a person whose
	// signature on a build --who vouches for.
	signerKey = "signer"
	// helpWidthKey is "help-width N": the most characters a line of help
	// may have.
	helpWidthKey = "help-width"
)

// settings are what pzm's own config says. With no config they are
// the defaults: nobody is vouched for and help is held to the default
// width.
type settings struct {
	path    string // where the config is, or would be
	signers who.Signers
	width   verb.Width
}

// settled is the result of reading the config, kept so it is read once.
type settled struct {
	set settings
	err error
}

// declared are the lines pzm reads. config.New cannot refuse these, as
// their keys are fixed here.
func declared() []config.Setting {
	signer, _ := config.New(signerKey, "a person whose signature on "+
		"a build --who vouches for", one, config.Tail)
	width, _ := config.New(helpWidthKey, "the most characters a line "+
		"of help may have", one, config.NoTail)
	return []config.Setting{signer, width}
}

// settings is what pzm's config says. It reads the config the
// first time and remembers what it found, or why it could not.
func (tool *Tool) settings() (settings, error) {
	if tool.settled == nil {
		set, err := readSettings(tool.machine)
		tool.settled = &settled{set, err}
	}
	return tool.settled.set, tool.settled.err
}

// readSettings reads pzm's own config from the machine, never a tool's. A
// config that is not there is the defaults. One that is there and does
// not conform is refused whole, every wrong line named by the file and
// the line.
func readSettings(machine host.Machine) (settings, error) {
	path, err := machine.ConfigPath()
	if err != nil {
		return settings{}, err
	}
	set := settings{path: path, width: verb.DefaultHelpWidth}
	raw, err := machine.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return settings{}, err
	}
	entries, err := config.Read(path, raw, declared()...)
	if err != nil {
		return settings{}, err
	}
	var refused []error
	seen := map[string]bool{}
	for _, each := range entries {
		if each.Key() == helpWidthKey && seen[helpWidthKey] {
			at := each.Line()
			twice := fmt.Errorf("%s:%d: %s is set twice", path, at,
				helpWidthKey)
			refused = append(refused, twice)
			continue
		}
		seen[each.Key()] = true
		if err := set.take(each); err != nil {
			at := each.Line()
			named := fmt.Errorf("%s:%d: %w", path, at, err)
			refused = append(refused, named)
		}
	}
	if err := errors.Join(refused...); err != nil {
		return settings{}, err
	}
	return set, nil
}

// take sets what one line of the config says. config has already seen to
// its key and its words, so the key is one of declared.
func (set *settings) take(each config.Entry) error {
	word := each.Words()[firstWord]
	if each.Key() == signerKey {
		return set.signers.Add(word, each.Tail())
	}
	number, err := strconv.Atoi(word)
	if err != nil || number <= none {
		return fmt.Errorf("%s is %q, not a positive whole number",
			helpWidthKey, word)
	}
	set.width = verb.Width(number)
	return nil
}

// helpRefusals is each line of help in the tool that is over width: a
// verb's, a verb's own flag's, or a flag of the tool.
func (tool *Tool) helpRefusals(width verb.Width) []error {
	var refused []error
	for _, each := range tool.flags {
		refused = append(refused, flagHelpRefusal(each, width)...)
	}
	for _, each := range tool.verbs {
		help := verb.Help(each.verb.Help())
		if err := verb.TooLong(help, width); err != nil {
			name := each.verb.Name()
			named := fmt.Errorf("verb %s: %w", name, err)
			refused = append(refused, named)
		}
		for _, own := range each.flags {
			more := flagHelpRefusal(own, width)
			refused = append(refused, more...)
		}
	}
	return refused
}

// flagHelpRefusal is why a flag's help is over width, or nothing.
func flagHelpRefusal(each Flag, width verb.Width) []error {
	help := verb.Help(each.declared.Help())
	err := verb.TooLong(help, width)
	if err == nil {
		return nil
	}
	long := each.declared.Long()
	return []error{fmt.Errorf("flag %s: %w", long, err)}
}
