package pzm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/janearc/puzzlemouth/config"
	"github.com/janearc/puzzlemouth/internal/who"
	"github.com/janearc/puzzlemouth/verb"
)

// pzmFile is what pzm's own config, ~/.config/pzm/config, may say. With
// no config nobody is vouched for and help is held to the default width.
type pzmFile struct {
	Signers []who.Signer `config:"signer,1,tail" help:"a person whose signature on a build --who vouches for"`
	Width   verb.Width   `config:"help-width,1" help:"the most characters a line of help may have"`
}

// schema is what pzmFile may say, made once.
var schema = must(config.NewSchema[pzmFile]())

// must is the schema of a declaration that is fixed in this file, which
// a test of this package holds sound. It panics on a refusal, since
// nothing a caller does can make the declaration right.
func must[File any](made *config.Schema[File],
	err error) *config.Schema[File] {
	if err != nil {
		panic(err)
	}
	return made
}

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
	set *settings
	err error
}

// settings is what pzm's config says. It reads the config the
// first time and remembers what it found, or why it could not.
func (tool *Tool) settings() (*settings, error) {
	if tool.settled == nil {
		set, err := readSettings()
		tool.settled = &settled{set, err}
	}
	return tool.settled.set, tool.settled.err
}

// configPath is ~/.config/pzm/config under the home the environment
// gives. It is not made. No tool's config is pzm's to open.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pzm", "config"), nil
}

// readSettings reads pzm's own config, never a tool's. A config that is
// not there is the defaults. One that is there and does not conform is
// refused whole, every wrong line named by the file and the line.
func readSettings() (*settings, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	set := &settings{path: path, width: verb.DefaultHelpWidth}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return set, nil
	}
	if err != nil {
		return nil, err
	}
	file := pzmFile{Width: verb.DefaultHelpWidth}
	if err := schema.Read(path, raw, &file); err != nil {
		return nil, err
	}
	vouched, err := vouchFor(path, file.Signers)
	if err != nil {
		return nil, err
	}
	set.signers, set.width = *vouched, file.Width
	return set, nil
}

// vouchFor is the signers of a config, or nil and every name that is
// set twice, each at its line. config has already seen to each key.
func vouchFor(path string, each []who.Signer) (*who.Signers, error) {
	var refused []error
	vouched := new(who.Signers)
	for _, signer := range each {
		if err := vouched.Add(signer.Name, signer.Key); err != nil {
			at := signer.At
			named := fmt.Errorf("%s:%d: signer: %w", path, at, err)
			refused = append(refused, named)
		}
	}
	if err := errors.Join(refused...); err != nil {
		return nil, err
	}
	return vouched, nil
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
