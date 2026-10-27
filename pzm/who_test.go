package pzm

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/verb"
	"go.uber.org/mock/gomock"
	"pgregory.net/rapid"
)

// perm is the mode the test files are made with. It's defined: the
// owner reads and writes, no one else does.
const perm = 0o600

const (
	// exitOK is the code of a call that did what it was asked. It's
	// defined by Tool.RunWith.
	exitOK = 0
	// exitFailed is the code of a verb that ran and failed. It's defined
	// by Tool.RunWith.
	exitFailed = 1
	// exitRefused is the code of a tool that could not run. It's defined
	// by Tool.RunWith.
	exitRefused = 2
)

// on is a tool whose machine is a mock with no config: the machine may
// be asked for it any number of times, and it is not there. Everything
// else the mock is asked is held to what a test expects: every call it
// was told to expect must have been made, and no other.
func on(t *testing.T) (*Tool, *MockMachine) {
	t.Helper()
	mock := NewMockMachine(gomock.NewController(t))
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return(configAt, nil).AnyTimes()
	mock.EXPECT().ReadFile(configAt).Return(nil, os.ErrNotExist).AnyTimes()
	return tool, mock
}

// onSigner is on, with a signer in the config that no signature here
// was made by.
func onSigner(t *testing.T) (*Tool, *MockMachine) {
	t.Helper()
	_, public := signedBinary(t)
	mock := NewMockMachine(gomock.NewController(t))
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return(configAt, nil).AnyTimes()
	configured := []byte("signer ada :: " + public + "\n")
	mock.EXPECT().ReadFile(configAt).Return(configured, nil).AnyTimes()
	return tool, mock
}

// binary is a file standing for the running binary, with the text
// beside it as its signature if there is any. It is the file's path.
func binary(t *testing.T, signature string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "demo")
	if err := os.WriteFile(path, []byte("a binary"), perm); err != nil {
		t.Fatal(err)
	}
	if signature != "" {
		beside := []byte(signature)
		if err := os.WriteFile(path+".sig", beside, perm); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// runWho runs --who on the tool and gives the exit code and the streams.
func runWho(tool *Tool) (int, string, string) {
	var out, errs bytes.Buffer
	code := tool.Run([]string{"--who"}, &out, &errs)
	return code, out.String(), errs.String()
}

// Anti-test: a signature beside the binary that does not check is said,
// and --who exits 1 rather than say who built it.
func TestABadSignatureIsSaid(t *testing.T) {
	tool, mock := onSigner(t)
	mock.EXPECT().Executable().Return(binary(t, "not a signature"), nil)
	code, out, errs := runWho(tool)
	if code != exitFailed || out != "" || !strings.Contains(errs,
		"demo who: ") || !strings.Contains(errs, "not an ssh signature") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: when the machine cannot say where the binary is, --who says
// so and exits 1. It does not go looking for a signature.
func TestAMachineThatCannotSayWhereTheBinaryIsFailsWho(t *testing.T) {
	tool, mock := onSigner(t)
	mock.EXPECT().Executable().Return("", errors.New("no path to demo"))
	code, out, errs := runWho(tool)
	if code != exitFailed || out != "" || !strings.Contains(errs,
		"demo who: no path to demo") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// With no signature beside the binary, --who says it is unsigned. It
// still says who built it.
func TestABinaryWithNoSignatureIsSaidUnsigned(t *testing.T) {
	tool, mock := onSigner(t)
	mock.EXPECT().Executable().Return(binary(t, ""), nil)
	code, out, errs := runWho(tool)
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo is not signed") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// The signature read is the one beside the path the machine gives. Here
// that is a file of the test's, and the real binary's is never looked at.
func TestTheSignatureReadIsBesideTheMachinesBinary(t *testing.T) {
	tool, mock := onSigner(t)
	mock.EXPECT().Executable().Return(binary(t, "still not one"), nil)
	if got, err := tool.signedBy(); err == nil || got.signer != "" {
		t.Errorf("a signature that is not one gave %q and %v", got.signer, err)
	}
}

// reads are the facts of a verb that only reads.
var reads = verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
	verb.ThisMachine, verb.NoOne, verb.Light)

// saysRan is a verb that writes down that it ran.
type saysRan struct{ ran *bool }

// do is the verb. It says it ran.
func (says saysRan) do(Call) error {
	*says.ran = true
	return nil
}

// barish is a tool on a mock machine whose bare call runs look. It
// also gives a flag that is set once look has run.
func barish(t *testing.T) (*Tool, *MockMachine, *bool) {
	t.Helper()
	tool, mock := on(t)
	ran := new(bool)
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", reads,
		saysRan{ran}.do)
	tool.Bare("look")
	return tool, mock, ran
}

// null opens /dev/null for reading, and a file of the test's for
// writing. They are two open files, which is all the machine is asked
// about; what is written can be read back with written.
func null(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	in, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	where := filepath.Join(t.TempDir(), "out")
	out, err := os.Create(where)
	if err != nil {
		t.Fatal(err)
	}
	return in, out
}

// written is what has been written to a file made by null.
func written(t *testing.T, out *os.File) string {
	t.Helper()
	text, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// At two terminals, a bare call runs the verb Bare named.
func TestABareCallAtTheMachinesTerminalRunsTheVerb(t *testing.T) {
	tool, mock, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	mock.EXPECT().IsTerminal(gomock.Any()).Return(true).Times(bothEnds)
	if code := tool.RunWith(nil, in, out, &bytes.Buffer{}); code != exitOK ||
		!*ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// bothEnds is how many files a call asks the machine about when it asks
// for the input and the output. It's defined.
const bothEnds = 2

// Anti-test: a bare call where the machine says output is no terminal
// prints the help and runs nothing.
func TestABareCallWhereTheMachineSaysNoTerminalRunsNothing(t *testing.T) {
	tool, mock, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	gomock.InOrder(
		mock.EXPECT().IsTerminal(gomock.Any()).Return(true),
		mock.EXPECT().IsTerminal(gomock.Any()).Return(false))
	code := tool.RunWith(nil, in, out, &bytes.Buffer{})
	said := written(t, out)
	if code != exitOK || *ran || !strings.HasPrefix(said, "demo: ") {
		t.Errorf("code %d, ran %v, said %q", code, *ran, said)
	}
}

// ends is one end of a call: a file the machine may call a terminal, or
// something that is no file and is never asked about.
type ends struct {
	file     bool
	terminal bool
}

// Both ends have to be terminals. The machine is asked about an end only
// if it is a file, and about the second only if the first said yes. The
// mock holds the code to exactly those questions.
func TestBothEndsMustBeTerminalsAndOnlyFilesAreAsked(t *testing.T) {
	rapid.Check(t, checkEnds)
}

// checkEnds is one draw of that property.
func checkEnds(t *rapid.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockMachine(ctrl)
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	first := ends{rapid.Bool().Draw(t, "in is a file"),
		rapid.Bool().Draw(t, "in is a terminal")}
	second := ends{rapid.Bool().Draw(t, "out is a file"),
		rapid.Bool().Draw(t, "out is a terminal")}
	in, out := open(t, first), open(t, second)
	defer closeEnd(in)
	defer closeEnd(out)
	if first.file {
		mock.EXPECT().IsTerminal(gomock.Any()).Return(first.terminal)
	}
	if first.file && first.terminal && second.file {
		mock.EXPECT().IsTerminal(gomock.Any()).Return(second.terminal)
	}
	want := first.file && first.terminal && second.file && second.terminal
	if got := tool.terminals(in, out); got != want {
		t.Fatalf("%+v %+v: got %v, want %v", first, second, got, want)
	}
	ctrl.Finish()
}

// open is an end of a call: a file, or a buffer.
func open(t *rapid.T, end ends) io.ReadWriter {
	if !end.file {
		return &bytes.Buffer{}
	}
	file, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

// closeEnd closes an end that is a file.
func closeEnd(end io.ReadWriter) {
	if file, isFile := end.(*os.File); isFile {
		file.Close()
	}
}

// kinds is every kind a flag has.
var kinds = []kind{switchKind, valueKind, listKind}

// The parser underneath asks a flag's value its type, and the type it
// gets is the kind's name.
func TestAFlagValuesTypeIsItsKindsName(t *testing.T) {
	rapid.Check(t, checkType)
}

// checkType is one draw of that property.
func checkType(t *rapid.T) {
	each := rapid.SampledFrom(kinds).Draw(t, "kind")
	value := &flagValue{flag: Flag{kind: each}}
	if value.Type() != each.String() {
		t.Fatalf("kind %v: type %q, name %q", each, value.Type(),
			each.String())
	}
}

// Anti-test: no two kinds of flag give the same type, so the parser can
// tell a switch from a value from a list.
func TestNoTwoKindsOfFlagShareAType(t *testing.T) {
	seen := map[string]kind{}
	for _, each := range kinds {
		value := &flagValue{flag: Flag{kind: each}}
		if other, twice := seen[value.Type()]; twice {
			t.Errorf("kinds %v and %v are both %q", other, each,
				value.Type())
		}
		seen[value.Type()] = each
	}
}

// Anti-test: with words after the tool, the machine is not asked whether
// there is a terminal. The mock has no expectations, so one question
// fails the test.
func TestWordsAfterTheToolMeanTheMachineIsNotAsked(t *testing.T) {
	tool, _, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith([]string{"look"}, in, out, &bytes.Buffer{})
	if code != exitOK || !*ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: a tool that named no bare verb never asks the machine about
// a terminal, and runs nothing bare.
func TestAToolWithNoBareVerbNeverAsksTheMachine(t *testing.T) {
	tool, _ := on(t)
	ran := new(bool)
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", reads,
		saysRan{ran}.do)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith(nil, in, out, &bytes.Buffer{})
	if code != exitOK || *ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: a tool that refuses itself runs nothing and does not ask
// the machine anything.
func TestAToolThatRefusesItselfNeverAsksTheMachine(t *testing.T) {
	tool, _, ran := barish(t)
	tool.Bare("nowhere")
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	var errs bytes.Buffer
	code := tool.RunWith(nil, in, out, &errs)
	if code != exitRefused || *ran ||
		!strings.Contains(errs.String(), "nowhere") {
		t.Errorf("code %d, ran %v, errs %q", code, *ran, errs.String())
	}
}

// Anti-test: --age is answered without the machine being asked about
// the binary or a terminal.
func TestAgeNeverAsksTheMachine(t *testing.T) {
	tool, _ := on(t)
	var out, errs bytes.Buffer
	code := tool.Run([]string{"--age"}, &out, &errs)
	if code != exitOK || errs.Len() != none {
		t.Errorf("code %d, errs %q", code, errs.String())
	}
}

// Anti-test: when the machine says the input is no terminal, it is not
// asked about the output.
func TestAnInputThatIsNoTerminalMeansTheOutputIsNotAsked(t *testing.T) {
	tool, mock, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	mock.EXPECT().IsTerminal(gomock.Any()).Return(false)
	code := tool.RunWith(nil, in, out, &bytes.Buffer{})
	if code != exitOK || *ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: an input that is no file is never put to the machine, and
// nothing runs bare.
func TestAnInputThatIsNoFileIsNeverPutToTheMachine(t *testing.T) {
	tool, _, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith(nil, &bytes.Buffer{}, out, &bytes.Buffer{})
	if code != exitOK || *ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// noHome is a home that is not there, so a tool that is not on a mock
// machine reads the config of no real person, and finds none.
const noHome = "/pzm-tests-have-no-home"

// TestMain gives the tests that home.
func TestMain(suite *testing.M) {
	os.Setenv("HOME", noHome)
	os.Exit(suite.Run())
}

// configAt is the path the mock machine gives for demo's config. It is
// relative, so it names no machine.
const configAt = "demo/.config/demo/config"

// withConfig is a tool on a mock machine whose config holds body. The
// mock expects the path to be asked for once and the file to be read
// once, and no more.
func withConfig(t *testing.T, body string) (*Tool, *MockMachine) {
	t.Helper()
	mock := NewMockMachine(gomock.NewController(t))
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return(configAt, nil)
	mock.EXPECT().ReadFile(configAt).Return([]byte(body), nil)
	return tool, mock
}

// signedBinary is a file signed for a build with a throwaway key made
// by ssh-keygen with no agent. It gives the file's path and the key's
// public line.
func signedBinary(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	key := filepath.Join(dir, "key")
	file := filepath.Join(dir, "demo")
	if err := os.WriteFile(file, []byte("a binary"), perm); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-q", "-t", "ed25519", "-N", "", "-C", "", "-f", key},
		{"-Y", "sign", "-n", "build", "-f", key, file}} {
		cmd := exec.Command("ssh-keygen", args...)
		cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("ssh-keygen %v: %v\n%s", args, err, out)
		}
	}
	public, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	return file, strings.TrimSpace(string(public))
}

// A configured signer is vouched for: --who says the signature was
// checked and names the signer.
func TestAConfiguredSignerIsVouchedFor(t *testing.T) {
	file, public := signedBinary(t)
	tool, mock := withConfig(t, "signer ada :: "+public+"\n")
	mock.EXPECT().Executable().Return(file, nil)
	code, out, errs := runWho(tool)
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo was signed by ada, as demo itself "+
			"checked;") || strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: with no config at all, no signer is set. --who says so
// plainly, says the binary is not signed even with a good signature
// beside it, and never asks the machine for the binary.
func TestWithNoConfigNoSignerIsSetAndNothingIsClaimed(t *testing.T) {
	mock := NewMockMachine(gomock.NewController(t))
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return(configAt, nil)
	mock.EXPECT().ReadFile(configAt).Return(nil, os.ErrNotExist)
	code, out, errs := runWho(tool)
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo is not signed") ||
		!strings.Contains(out, "no signer is set: no key is vouched "+
			"for; add a signer line to "+configAt) ||
		strings.Contains(out, "signed by") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: a config with other settings and no signer is the same as
// none. A signature beside the binary is not read.
func TestAConfigWithNoSignerLineSetsNoSigner(t *testing.T) {
	tool, _ := withConfig(t, "# nothing here\nhelp-width 60\n")
	code, out, _ := runWho(tool)
	if code != exitOK || !strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q", code, out)
	}
}

// Anti-test: a stranger's key is not vouched for, as before. The
// signature stays an error that names the unknown key, with a signer
// configured.
func TestAStrangersKeyIsStillRefusedWithASignerSet(t *testing.T) {
	file, _ := signedBinary(t)
	_, other := signedBinary(t)
	tool, mock := withConfig(t, "signer ada :: "+other+"\n")
	mock.EXPECT().Executable().Return(file, nil)
	code, out, errs := runWho(tool)
	if code != exitFailed || out != "" ||
		!strings.Contains(errs, "does not know") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: a signature that does not check is still said, with a
// signer configured.
func TestABadSignatureIsStillSaidWithASignerSet(t *testing.T) {
	_, public := signedBinary(t)
	tool, mock := withConfig(t, "signer ada :: "+public+"\n")
	mock.EXPECT().Executable().Return(binary(t, "not a signature"), nil)
	code, _, errs := runWho(tool)
	if code != exitFailed || !strings.Contains(errs, "not an ssh signature") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// Anti-test: with a signer set and no signature beside the binary, the
// binary is not signed, and no line says a signer is missing.
func TestASignerSetAndNoSignatureIsUnsigned(t *testing.T) {
	_, public := signedBinary(t)
	tool, mock := withConfig(t, "signer ada :: "+public+"\n")
	mock.EXPECT().Executable().Return(binary(t, ""), nil)
	code, out, _ := runWho(tool)
	if code != exitOK || !strings.Contains(out, "demo is not signed") ||
		strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q", code, out)
	}
}

// refusedByConfig is a config body that is refused, and what the
// refusal must name.
type refusedByConfig struct{ why, body, names string }

// Anti-test: a config that cannot be read is refused whole, and the
// refusal names the file and the line.
func TestAMalformedConfigIsRefusedWithTheLineNamed(t *testing.T) {
	_, public := signedBinary(t)
	cases := []refusedByConfig{
		{"a signer line with no key", "signer ada :: ssh-ed25519",
			configAt + ":1: signer ada"},
		{"a signer key that is not base64",
			"# c\nsigner ada :: ssh-ed25519 !!!", configAt + ":2: signer ada"},
		{"a signer with no tail", "signer ada", configAt + ":1:"},
		{"a signer named twice", "signer ada :: " + public +
			"\nsigner ada :: " + public, configAt + ":2: signer ada is set twice"},
		{"a line that is no setting", "signer ada :: " + public +
			"\nwho-am-i ada", configAt + ":2:"},
	}
	for _, each := range cases {
		tool, _ := withConfig(t, each.body)
		code, out, errs := runWho(tool)
		if code != exitRefused || out != "" ||
			!strings.Contains(errs, each.names) {
			t.Errorf("%s: code %d, out %q, errs %q", each.why, code, out,
				errs)
		}
	}
}

// Anti-test: a config that cannot be found or read is refused, and the
// refusal says why. Only a file that is not there means no config.
func TestAConfigThatCannotBeReachedIsRefused(t *testing.T) {
	mock := NewMockMachine(gomock.NewController(t))
	tool := New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return("", errors.New("no home"))
	if code, _, errs := runWho(tool); code != exitRefused ||
		!strings.Contains(errs, "no home") {
		t.Errorf("code %d, errs %q", code, errs)
	}
	mock = NewMockMachine(gomock.NewController(t))
	tool = New("demo", "a tool for testing pzm", "", "", "", "")
	tool.machine = mock
	mock.EXPECT().ConfigPath("demo").Return(configAt, nil)
	mock.EXPECT().ReadFile(configAt).Return(nil, os.ErrPermission)
	if code, _, errs := runWho(tool); code != exitRefused ||
		!strings.Contains(errs, "permission") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// The signer is looked for once: a second ask reads nothing again. The
// mock expects one read.
func TestTheConfigIsReadOnce(t *testing.T) {
	tool, _ := withConfig(t, "")
	for range twice {
		if _, err := tool.signedBy(); err != nil {
			t.Fatal(err)
		}
	}
}

// helpOf is a line of help that long.
func helpOf(length int) verb.Help {
	return verb.Help(strings.Repeat("x", length))
}

// ranNothing is a verb that does nothing.
func ranNothing(Call) error { return nil }

const (
	// widest is the most characters help may have by default. It's
	// defined by verb.DefaultHelpWidth.
	widest = int(verb.DefaultHelpWidth)
	// hundred is a width a config sets, wider than the default. It's
	// defined here.
	hundred = 100
	// twenty is a width a config sets, narrower than the default. It's
	// defined here.
	twenty = 20
	// small is the length of help far inside any width. It's defined
	// here.
	small = 5
	// twice is how many times a test asks for the same thing. It's
	// defined here.
	twice = 2
)

// declaredWith is a tool on a mock with config body that declares a
// verb with n characters of help and a flag with m.
func declaredWith(t *testing.T, body string, verbHelp,
	flagHelp int) *Tool {
	t.Helper()
	tool, _ := withConfig(t, body)
	flagText, verbText := helpOf(flagHelp), helpOf(verbHelp)
	tool.Flags(Switch("everywhere", 'e', flagText))
	tool.Verb("look", 'l', verbText, "# look", "ExampleLook", reads,
		ranNothing, Switch("loud", 'o', flagText))
	return tool
}

// With nothing configured, help is held to the default: 70 is kept and
// 71 is refused, for a verb, for a flag of its own and for a flag of
// the tool.
func TestWithNoWidthConfiguredHelpIsHeldToTheDefault(t *testing.T) {
	if err := declaredWith(t, "", widest, widest).Check(); err != nil {
		t.Errorf("help of exactly the default: %v", err)
	}
	err := declaredWith(t, "", widest+one, widest).Check()
	if err == nil || !strings.Contains(err.Error(), "verb look: help: "+
		"it is over 70 characters") {
		t.Errorf("a verb over the default: %v", err)
	}
	err = declaredWith(t, "", widest, widest+one).Check()
	if err == nil || !strings.Contains(err.Error(), "flag --loud: help: ") ||
		!strings.Contains(err.Error(), "flag --everywhere: help: ") {
		t.Errorf("flags over the default: %v", err)
	}
}

// A help-width line sets the width: wider keeps what the default would
// refuse, and narrower refuses what the default would keep.
func TestAHelpWidthLineSetsTheWidth(t *testing.T) {
	wide := "help-width 100\n"
	if err := declaredWith(t, wide, hundred, hundred).Check(); err != nil {
		t.Errorf("help of 100 under a width of 100: %v", err)
	}
	err := declaredWith(t, wide, hundred+one, hundred).Check()
	if err == nil || !strings.Contains(err.Error(), "over 100 characters") {
		t.Errorf("101 under a width of 100: %v", err)
	}
	narrow := "# narrow\nhelp-width 20\n"
	err = declaredWith(t, narrow, twenty+one, small).Check()
	if err == nil || !strings.Contains(err.Error(), "over 20 characters") {
		t.Errorf("21 under a width of 20: %v", err)
	}
	if err := declaredWith(t, narrow, twenty, twenty).Check(); err != nil {
		t.Errorf("20 under a width of 20: %v", err)
	}
}

// Anti-test: a width that is not a positive whole number is refused with
// the line named, and so is one set twice.
func TestAWidthThatIsNotAPositiveWholeNumberIsRefused(t *testing.T) {
	for _, each := range []refusedByConfig{
		{"zero", "help-width 0", configAt + ":1: help-width"},
		{"negative", "\nhelp-width -3", configAt + ":2: help-width"},
		{"fraction", "help-width 7.5", configAt + ":1: help-width"},
		{"words", "help-width wide", configAt + ":1: help-width"},
		{"too many for an int", "help-width 99999999999999999999",
			configAt + ":1: help-width"},
		{"twice", "help-width 5\nhelp-width 6",
			configAt + ":2: help-width is set twice"},
	} {
		tool, _ := withConfig(t, each.body)
		err := tool.Check()
		if err == nil || !strings.Contains(err.Error(), each.names) {
			t.Errorf("%s: %v", each.why, err)
		}
	}
}

// A refused width stops the tool: nothing runs, and it exits 2.
func TestARefusedWidthStopsTheTool(t *testing.T) {
	tool, _ := withConfig(t, "help-width 0")
	var out, errs bytes.Buffer
	if code := tool.Run([]string{"--age"}, &out, &errs); code != exitRefused ||
		out.Len() != none || !strings.Contains(errs.String(), configAt+":1:") {
		t.Errorf("code %d, out %q, errs %q", code, out.String(), errs.String())
	}
}

// Anti-test: asked directly, with no Check before it, signedBy says a
// config that is refused and claims no signer.
func TestSignedByWithARefusedConfigClaimsNoSigner(t *testing.T) {
	tool, _ := withConfig(t, "signer ada")
	got, err := tool.signedBy()
	if err == nil || got.signer != "" || got.unset ||
		!strings.Contains(err.Error(), configAt+":1:") {
		t.Errorf("got %+v, %v", got, err)
	}
}
