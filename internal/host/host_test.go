package host_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/janearc/puzzlemouth/internal/host"
	"pgregory.net/rapid"
)

// perm is the mode the test files are made with. It's defined: the
// owner reads and writes, no one else does.
const perm = 0o600

// terminal is whether the local machine calls an open file a terminal.
func terminal(file *os.File) bool {
	fd := int(file.Fd())
	return host.NewLocal().IsTerminal(fd)
}

// The local machine's binary is the one running, and it is there to be
// found.
func TestTheLocalMachinesBinaryIsTheRunningOne(t *testing.T) {
	path, err := host.NewLocal().Executable()
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.Executable()
	if err != nil || path != want {
		t.Errorf("got %q, want %q (%v)", path, want, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the binary %q is not there: %v", path, err)
	}
}

// Anti-test: the local machine does not call a file a terminal. It asks
// the open descriptor, so the text in the file cannot change the answer.
// The drawn text varies the file, and the property holds for all of it.
func TestTheLocalMachineDoesNotCallAFileATerminal(t *testing.T) {
	dir := t.TempDir()
	rapid.Check(t, files{dir}.check)
}

// files are the files a property is drawn with, made in dir.
type files struct{ dir string }

// check is one draw of that property. It writes some text to a file,
// opens it and asks the machine about it.
func (made files) check(t *rapid.T) {
	text := rapid.String().Draw(t, "text")
	path := filepath.Join(made.dir, "file")
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if terminal(file) {
		t.Fatalf("a file holding %q was called a terminal", text)
	}
}

// Anti-test: the null device is no terminal either.
func TestTheNullDeviceIsNoTerminal(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if terminal(null) {
		t.Error("the null device was called a terminal")
	}
}

// pzm's own config is ~/.config/pzm/config, under the home the
// environment gives.
func TestPzmsConfigIsUnderTheHomesDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	got, err := host.NewLocal().ConfigPath()
	want := filepath.Join(home, ".config", "pzm", "config")
	if err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}

// Anti-test: with no home to be found, there is no config path and the
// error says so. It does not guess a directory.
func TestWithNoHomeThereIsNoConfigPath(t *testing.T) {
	t.Setenv("HOME", "")
	got, err := host.NewLocal().ConfigPath()
	if err == nil || got != "" {
		t.Errorf("got %q, %v", got, err)
	}
}

// The local machine reads a file whole, and a file that is not there is
// an error that says it is not.
func TestTheLocalMachineReadsAFileOrSaysItIsNotThere(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("signer"), perm); err != nil {
		t.Fatal(err)
	}
	body, err := host.NewLocal().ReadFile(path)
	if err != nil || string(body) != "signer" {
		t.Errorf("got %q, %v", body, err)
	}
	_, err = host.NewLocal().ReadFile(path + ".gone")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a file not there: %v", err)
	}
}
