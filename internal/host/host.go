// Package host asks the machine a program runs on, so a mock can answer.
package host

import (
	"os"
	"path/filepath"

	"golang.org/x/term"
)

// Machine is what pzm asks of the machine a tool runs on.
type Machine interface {
	// Executable is the path of the running binary.
	Executable() (string, error)
	// IsTerminal is whether the open file descriptor is a terminal.
	IsTerminal(fd int) bool
	// ConfigPath is where pzm keeps its own config: the file
	// ~/.config/pzm/config. It is not made. No tool's config is pzm's
	// to open.
	ConfigPath() (string, error)
	// ReadFile is the whole of the file at path. A file that is not there
	// is an error that errors.Is os.ErrNotExist.
	ReadFile(path string) ([]byte, error)
}

// local is the machine the program is running on.
type local struct{}

// NewLocal is the machine the program is running on.
func NewLocal() Machine { return local{} }

// Executable is the path of the running binary.
func (local) Executable() (string, error) { return os.Executable() }

// IsTerminal is whether the open file descriptor is a terminal.
func (local) IsTerminal(fd int) bool { return term.IsTerminal(fd) }

// ConfigPath is ~/.config/pzm/config under the home the environment
// gives.
func (local) ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pzm", "config"), nil
}

// ReadFile is the whole of the file at path.
func (local) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
