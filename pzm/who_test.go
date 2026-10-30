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
	"pgregory.net/rapid"
)

// perm is the mode the test files are made with. It's defined: the
// owner reads and writes, no one else does.
const perm = 0o600

// dirPerm is the mode the test directories are made with. It's defined:
// the owner alone enters them.
const dirPerm = 0o700

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

// demo is a tool for the tests to run.
func demo() *Tool {
	return New("demo", "a tool for testing pzm", "", "", "", "")
}

// configIn makes a temporary home and gives the path pzm's config would
// be at under it. The file is not made.
func configIn(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return filepath.Join(home, ".config", "pzm", "config")
}

// on is a tool whose home is a temporary directory with no config in it.
func on(t *testing.T) *Tool {
	t.Helper()
	configIn(t)
	return demo()
}

// withConfig is a tool whose home holds a pzm config with body in it, and
// the path of that config.
func withConfig(t *testing.T, body string) (*Tool, string) {
	t.Helper()
	path := configIn(t)
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), perm); err != nil {
		t.Fatal(err)
	}
	return demo(), path
}

// onSigner is a tool whose config has a signer that no signature here
// was made by.
func onSigner(t *testing.T) *Tool {
	t.Helper()
	_, public := newKey(t)
	tool, _ := withConfig(t, "signer ada :: "+public+"\n")
	return tool
}

// running is the path of the running binary, which is the test binary.
// The signature that --who reads is beside it, so a test that puts one
// there has it taken away when the test is done.
func running(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { takeAway(t, path+".sig") })
	return path
}

// takeAway removes a file a test made, if it is still there.
func takeAway(t *testing.T, path string) {
	t.Helper()
	err := os.Remove(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Error(err)
	}
}

// signatureBeside puts text beside the running binary as its signature.
func signatureBeside(t *testing.T, text string) {
	t.Helper()
	path := running(t) + ".sig"
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
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
	tool := onSigner(t)
	signatureBeside(t, "not a signature")
	code, out, errs := runWho(tool)
	if code != exitFailed || out != "" || !strings.Contains(errs,
		"demo who: ") || !strings.Contains(errs, "not an ssh signature") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// With no signature beside the binary, --who says it is unsigned. It
// still says who built it.
func TestABinaryWithNoSignatureIsSaidUnsigned(t *testing.T) {
	tool := onSigner(t)
	running(t)
	code, out, errs := runWho(tool)
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo is not signed") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// The signature read is the one beside the running binary, whatever is
// beside any other.
func TestTheSignatureReadIsBesideTheRunningBinary(t *testing.T) {
	tool := onSigner(t)
	signatureBeside(t, "still not one")
	tool.executable = running(t)
	if got, err := tool.signedBy(); err == nil || got != nil {
		t.Errorf("a signature that is not one gave %+v and %v", got, err)
	}
}

// Anti-test: a tool that cannot tell where it is running is refused at
// the edge with exit 2, and runs nothing.
func TestAToolThatCannotTellWhereItIsRunningIsRefusedAtTheEdge(t *testing.T) {
	tool, ran := barish(t)
	var out, errs bytes.Buffer
	code := tool.start([]string{"look"}, &bytes.Buffer{}, &out, &errs, "",
		errors.New("no path to demo"))
	if code != exitRefused || *ran || out.Len() != none ||
		!strings.Contains(errs.String(), "demo: cannot tell where it is "+
			"running: no path to demo") {
		t.Errorf("code %d, ran %v, errs %q", code, *ran, errs.String())
	}
}

// What the edge found is handed to the tool as plain values: the path it
// was given is the one --who looks beside.
func TestTheEdgeHandsTheToolThePathItFound(t *testing.T) {
	tool := on(t)
	path := running(t)
	var out, errs bytes.Buffer
	code := tool.start([]string{"--age"}, &bytes.Buffer{}, &out, &errs,
		path, nil)
	if code != exitOK || tool.executable != path || tool.terminal {
		t.Errorf("code %d, path %q, terminal %v", code, tool.executable,
			tool.terminal)
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

// barish is a tool whose bare call runs look. It also gives a flag that
// is set once look has run. No test here has a terminal, so it asks the
// open files and finds none.
func barish(t *testing.T) (*Tool, *bool) {
	t.Helper()
	tool := on(t)
	ran := new(bool)
	tool.Verb("look", 'l', "looks", "# look", "ExampleLook", reads,
		saysRan{ran}.do)
	tool.Bare("look")
	return tool, ran
}

// null opens /dev/null for reading, and a file of the test's for
// writing. They are two open files; what is written can be read back with
// written.
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
func TestABareCallAtTwoTerminalsRunsTheVerb(t *testing.T) {
	tool, ran := barish(t)
	tool.AtTerminal(atATerminal)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	if code := tool.RunWith(nil, in, out, &bytes.Buffer{}); code != exitOK ||
		!*ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: a bare call where there is no terminal prints the help and
// runs nothing.
func TestABareCallWhereThereIsNoTerminalRunsNothing(t *testing.T) {
	tool, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith(nil, in, out, &bytes.Buffer{})
	said := written(t, out)
	if code != exitOK || *ran || !strings.HasPrefix(said, "demo: ") {
		t.Errorf("code %d, ran %v, said %q", code, *ran, said)
	}
}

// atATerminal says every call is at a terminal. The tests need it, since
// their input and output never are one.
func atATerminal(io.Reader, io.Writer) bool { return true }

// ends is one end of a call: an open file, or something that is no file.
type ends struct{ file bool }

// Anti-test: no test here has a terminal, so no end of a call is one,
// whether it is an open file such as /dev/null or no file at all.
func TestNoEndOfATestsCallIsATerminal(t *testing.T) {
	rapid.Check(t, checkEnds)
}

// checkEnds is one draw of that property.
func checkEnds(t *rapid.T) {
	first := ends{rapid.Bool().Draw(t, "in is a file")}
	second := ends{rapid.Bool().Draw(t, "out is a file")}
	in, out := open(t, first), open(t, second)
	defer closeEnd(in)
	defer closeEnd(out)
	if terminals(in, out) {
		t.Fatalf("%+v %+v: taken for terminals", first, second)
	}
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

// With words after the tool, the verb they name runs, bare or not.
func TestWordsAfterTheToolRunTheVerbTheyName(t *testing.T) {
	tool, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith([]string{"look"}, in, out, &bytes.Buffer{})
	if code != exitOK || !*ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: a tool that named no bare verb runs nothing bare.
func TestAToolWithNoBareVerbRunsNothingBare(t *testing.T) {
	tool := on(t)
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

// Anti-test: a tool that refuses itself runs nothing.
func TestAToolThatRefusesItselfRunsNothing(t *testing.T) {
	tool, ran := barish(t)
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

// Anti-test: --age is answered with no binary and no terminal in it.
func TestAgeIsAnswered(t *testing.T) {
	tool := on(t)
	var out, errs bytes.Buffer
	code := tool.Run([]string{"--age"}, &out, &errs)
	if code != exitOK || errs.Len() != none {
		t.Errorf("code %d, errs %q", code, errs.String())
	}
}

// Anti-test: an input that is a file and no terminal means nothing runs
// bare.
func TestAnInputThatIsNoTerminalRunsNothingBare(t *testing.T) {
	tool, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith(nil, in, out, &bytes.Buffer{})
	if code != exitOK || *ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// Anti-test: an input that is no file means nothing runs bare.
func TestAnInputThatIsNoFileRunsNothingBare(t *testing.T) {
	tool, ran := barish(t)
	in, out := null(t)
	defer in.Close()
	defer out.Close()
	code := tool.RunWith(nil, &bytes.Buffer{}, out, &bytes.Buffer{})
	if code != exitOK || *ran {
		t.Errorf("code %d, ran %v", code, *ran)
	}
}

// noHome is a home that is not there, so a test that does not make its own
// reads the config of no real person, and finds none.
const noHome = "/pzm-tests-have-no-home"

// TestMain gives the tests that home.
func TestMain(suite *testing.M) {
	os.Setenv("HOME", noHome)
	os.Exit(suite.Run())
}

// newKey makes a throwaway ed25519 key with ssh-keygen, with no agent. It
// gives the key's path and its public line.
func newKey(t *testing.T) (string, string) {
	t.Helper()
	key := filepath.Join(t.TempDir(), "key")
	keygen(t, "-q", "-t", "ed25519", "-N", "", "-C", "", "-f", key)
	public, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	return key, strings.TrimSpace(string(public))
}

// keygen runs ssh-keygen with no agent, so a test never reaches a real
// key.
func keygen(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("ssh-keygen", args...)
	cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen %v: %v\n%s", args, err, out)
	}
}

// signRunning signs the running binary for a build with the key, leaving
// the signature beside it until the test is done.
func signRunning(t *testing.T, key string) {
	t.Helper()
	path := running(t)
	keygen(t, "-Y", "sign", "-n", "build", "-f", key, path)
}

// signedBy signs the running binary with a throwaway key and gives that
// key's public line.
func signedBy(t *testing.T) string {
	t.Helper()
	key, public := newKey(t)
	signRunning(t, key)
	return public
}

// A configured signer is vouched for: --who says the signature was
// checked and names the signer.
func TestAConfiguredSignerIsVouchedFor(t *testing.T) {
	tool, _ := withConfig(t, "signer ada :: "+signedBy(t)+"\n")
	code, out, errs := runWho(tool)
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo was signed by ada, as demo itself "+
			"checked;") || strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: with no config at all, no signer is set. --who says so
// plainly, says the binary is not signed even with a good signature
// beside it.
func TestWithNoConfigNoSignerIsSetAndNothingIsClaimed(t *testing.T) {
	path := configIn(t)
	signedBy(t)
	code, out, errs := runWho(demo())
	if code != exitOK || errs != "" ||
		!strings.Contains(out, "demo is not signed") ||
		!strings.Contains(out, "no signer is set: no key is vouched "+
			"for; add a signer line to "+path) ||
		strings.Contains(out, "signed by") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: a config with other settings and no signer is the same as
// none. A signature beside the binary is not read.
func TestAConfigWithNoSignerLineSetsNoSigner(t *testing.T) {
	tool, _ := withConfig(t, "# nothing here\nhelp-width 60\n")
	signatureBeside(t, "not a signature")
	code, out, _ := runWho(tool)
	if code != exitOK || !strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q", code, out)
	}
}

// Anti-test: a stranger's key is not vouched for, as before. The
// signature stays an error that names the unknown key, with a signer
// configured.
func TestAStrangersKeyIsStillRefusedWithASignerSet(t *testing.T) {
	signedBy(t)
	_, other := newKey(t)
	tool, _ := withConfig(t, "signer ada :: "+other+"\n")
	code, out, errs := runWho(tool)
	if code != exitFailed || out != "" ||
		!strings.Contains(errs, "does not know") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: a signature that does not check is still said, with a
// signer configured.
func TestABadSignatureIsStillSaidWithASignerSet(t *testing.T) {
	_, public := newKey(t)
	tool, _ := withConfig(t, "signer ada :: "+public+"\n")
	signatureBeside(t, "not a signature")
	code, _, errs := runWho(tool)
	if code != exitFailed || !strings.Contains(errs, "not an ssh signature") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// Anti-test: with a signer set and no signature beside the binary, the
// binary is not signed, and no line says a signer is missing.
func TestASignerSetAndNoSignatureIsUnsigned(t *testing.T) {
	_, public := newKey(t)
	tool, _ := withConfig(t, "signer ada :: "+public+"\n")
	running(t)
	code, out, _ := runWho(tool)
	if code != exitOK || !strings.Contains(out, "demo is not signed") ||
		strings.Contains(out, "no signer is set") {
		t.Errorf("code %d, out %q", code, out)
	}
}

// refusedByConfig is a config body that is refused, and what the
// refusal must hold after the config's path.
type refusedByConfig struct{ why, body, names string }

// Anti-test: a config that cannot be read is refused whole, and the
// refusal names the file, the line and the key.
func TestAMalformedConfigIsRefusedWithTheLineNamed(t *testing.T) {
	_, public := newKey(t)
	cases := []refusedByConfig{
		{"a signer line with no key", "signer ada :: ssh-ed25519",
			":1: signer: "},
		{"a signer key that is not base64",
			"# c\nsigner ada :: ssh-ed25519 !!!",
			":2: signer: the key is not base64"},
		{"a signer with no tail", "signer ada", ":1: signer: "},
		{"a signer named twice", "signer ada :: " + public +
			"\nsigner ada :: " + public,
			":2: signer: signer ada is set twice"},
		{"a line that is no setting", "signer ada :: " + public +
			"\nwho-am-i ada", ":2: who-am-i: "},
	}
	for _, each := range cases {
		tool, path := withConfig(t, each.body)
		code, out, errs := runWho(tool)
		if code != exitRefused || out != "" ||
			!strings.Contains(errs, path+each.names) {
			t.Errorf("%s: code %d, out %q, errs %q", each.why, code, out,
				errs)
		}
	}
}

// Anti-test: every refusal in a config is reported together, and the
// path is named for each.
func TestEveryRefusalInAConfigIsReportedTogether(t *testing.T) {
	_, public := newKey(t)
	body := "signer ada :: " + public + "\nsigner ada :: " + public +
		"\nsigner bo :: " + public + "\nsigner bo :: " + public + "\n"
	tool, path := withConfig(t, body)
	err := tool.Check()
	if err == nil || !strings.Contains(err.Error(), path+":2: signer: ") ||
		!strings.Contains(err.Error(), path+":4: signer: ") {
		t.Errorf("check: %v", err)
	}
}

// Anti-test: a config that cannot be found or read is refused, and the
// refusal says why. Only a file that is not there means no config.
func TestAConfigThatCannotBeReachedIsRefused(t *testing.T) {
	t.Setenv("HOME", "")
	if code, _, errs := runWho(demo()); code != exitRefused ||
		!strings.Contains(errs, "HOME") {
		t.Errorf("code %d, errs %q", code, errs)
	}
	path := configIn(t)
	if err := os.MkdirAll(path, dirPerm); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := runWho(demo()); code != exitRefused ||
		!strings.Contains(errs, "directory") {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// The config is read once: a second ask reads nothing again, so a file
// changed between the two is not read.
func TestTheConfigIsReadOnce(t *testing.T) {
	tool, path := withConfig(t, "")
	for range twice {
		if _, err := tool.signedBy(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("bogus\n"), perm); err != nil {
			t.Fatal(err)
		}
	}
}

// The declaration of pzm's own file is sound, and a refusal of one
// would stop the program.
func TestMustStopsOnARefusedDeclaration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a refused declaration did not stop the program")
		}
	}()
	must[pzmFile](nil, errors.New("refused"))
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

// declaredWith is a tool with config body that declares a verb with n
// characters of help and a flag with m.
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
		{"zero", "help-width 0", ":1: help-width: "},
		{"negative", "\nhelp-width -3", ":2: help-width: "},
		{"fraction", "help-width 7.5", ":1: help-width: "},
		{"words", "help-width wide", ":1: help-width: "},
		{"too many for an int", "help-width 99999999999999999999",
			":1: help-width: "},
		{"twice", "help-width 5\nhelp-width 6", ":2: help-width: "},
	} {
		tool, path := withConfig(t, each.body)
		err := tool.Check()
		if err == nil || !strings.Contains(err.Error(), path+each.names) {
			t.Errorf("%s: %v", each.why, err)
		}
	}
}

// A refused width stops the tool: nothing runs, and it exits 2.
func TestARefusedWidthStopsTheTool(t *testing.T) {
	tool, path := withConfig(t, "help-width 0")
	var out, errs bytes.Buffer
	if code := tool.Run([]string{"--age"}, &out, &errs); code != exitRefused ||
		out.Len() != none || !strings.Contains(errs.String(), path+":1:") {
		t.Errorf("code %d, out %q, errs %q", code, out.String(), errs.String())
	}
}

// Anti-test: asked directly, with no Check before it, signedBy says a
// config that is refused and claims no signer.
func TestSignedByWithARefusedConfigClaimsNoSigner(t *testing.T) {
	tool, path := withConfig(t, "signer ada")
	got, err := tool.signedBy()
	if err == nil || got != nil ||
		!strings.Contains(err.Error(), path+":1:") {
		t.Errorf("got %+v, %v", got, err)
	}
}

// foreign is a tool's own config as game keeps one: lines pzm does not
// know, and a help-width that would be refused if pzm read it.
const foreign = "author ada\nlint on\nproject x {\n}\nhelp-width 0\n"

// Anti-test: a tool's config under HOME full of foreign lines changes
// nothing: --who runs and Check is clean. If pzm read it, the help-width
// line would be refused.
func TestAForeignConfigOnDiskChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "demo")
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(foreign), perm); err != nil {
		t.Fatal(err)
	}
	tool := demo()
	if err := tool.Check(); err != nil {
		t.Errorf("check: %v", err)
	}
	code, _, errs := runWho(tool)
	if code != exitOK || errs != "" {
		t.Errorf("code %d, errs %q", code, errs)
	}
}

// pzm's own file on the real machine is ~/.config/pzm/config: a signer
// there is read, and a bad line there is refused with file and line.
func TestPzmsOwnFileOnDisk(t *testing.T) {
	tool, path := withConfig(t, "help-width 90\nauthor x\n")
	err := tool.Check()
	if err == nil || !strings.Contains(err.Error(), path+":2:") {
		t.Errorf("check: %v", err)
	}
}

// With no pzm file there are the defaults.
func TestWithNoPzmFileThereAreDefaults(t *testing.T) {
	if err := on(t).Check(); err != nil {
		t.Errorf("check: %v", err)
	}
}
