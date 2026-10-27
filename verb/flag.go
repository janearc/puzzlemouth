package verb

import (
	"errors"
	"fmt"
)

// Flag is one flag a verb or a tool declares. Its name is its long flag,
// and it has a short flag only where one earns its letter. It has one
// line of help. How it takes a value is the tool's to say.
type Flag struct {
	name  Name
	short Short
	help  Help
}

// NewFlag is a flag, or why it cannot be one. Every refusal comes at
// once, each naming the part it is about. The rules are those a verb's
// name, short flag and help are held to.
// Help is held to width, as a verb's is by NewWithin.
func NewFlag(width Width, name Name, short Short, help Help) (Flag,
	error) {
	refused := partsRefused(width, name, short, help)
	if len(refused) > none {
		return Flag{}, fmt.Errorf("flag --%s: %w", name,
			errors.Join(refused...))
	}
	return Flag{name, short, help}, nil
}

// Name is the flag's name.
func (flag Flag) Name() string { return string(flag.name) }

// Long is the flag as it is given: its name after two hyphens.
func (flag Flag) Long() string { return "--" + string(flag.name) }

// Short is the flag's short form after one hyphen, or nothing when it has
// none.
func (flag Flag) Short() string {
	if flag.short == NoShort {
		return ""
	}
	return "-" + string(rune(flag.short))
}

// Help is the flag's line for --help.
func (flag Flag) Help() string { return string(flag.help) }
