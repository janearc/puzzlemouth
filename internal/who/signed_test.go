package who

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/janearc/puzzlemouth/config"
	"pgregory.net/rapid"
)

// keygen runs ssh-keygen with no agent, so a test never reaches a real
// key or asks anyone for Touch ID.
func keygen(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("ssh-keygen", args...)
	cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// signed makes a throwaway key and a file signed with it for space. Any
// options go on to ssh-keygen. It gives the file's path and the key's
// public line.
func signed(t *testing.T, space string, options ...string) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Fatal("no ssh-keygen to sign with: this check cannot run")
	}
	dir := t.TempDir()
	key := filepath.Join(dir, "key")
	keygen(t, "-q", "-t", "ed25519", "-N", "", "-C", "", "-f", key)
	file := filepath.Join(dir, "gort")
	if err := os.WriteFile(file, []byte("a binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	args := append([]string{"-Y", "sign", "-n", space, "-f", key}, options...)
	keygen(t, append(args, file)...)
	public, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	return file, strings.TrimSpace(string(public))
}

// rewritten lets a test damage one part of the signature beside file. It
// decodes the signature and changes it. Then it writes it back armoured.
func rewritten(t *testing.T, file string, change func([]byte) []byte) {
	t.Helper()
	armour, err := os.ReadFile(file + ".sig")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.TrimSpace(string(armour))
	body = strings.TrimSuffix(strings.TrimPrefix(body, armourTop),
		armourBottom)
	raw, err := base64.StdEncoding.DecodeString(
		strings.Join(strings.Fields(body), ""))
	if err != nil {
		t.Fatal(err)
	}
	out := armourTop + "\n" +
		base64.StdEncoding.EncodeToString(change(raw)) + "\n" +
		armourBottom + "\n"
	if err := os.WriteFile(file+".sig", []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSignerVerifies(t *testing.T) {
	for _, options := range [][]string{nil, {"-O", "hashalg=sha256"}} {
		file, public := signed(t, "build", options...)
		got, err := signerAmong(file, map[string]string{"tester": public})
		if err != nil || got != "tester" {
			t.Errorf("signed with %q: got %q, %v; want tester",
				options, got, err)
		}
	}
}

func TestSignerWithoutSignature(t *testing.T) {
	file := filepath.Join(t.TempDir(), "gort")
	got, err := signerAmong(file, map[string]string{})
	if err != nil || got != "" {
		t.Fatalf("got %q, %v; want nobody and no error", got, err)
	}
}

func TestSignerRefuses(t *testing.T) {
	file, _ := signed(t, "build")
	other, stranger := signed(t, "build")
	gitFile, gitPublic := signed(t, "git")
	if err := os.WriteFile(other, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, file, public, want string
	}{
		{"changed after signing", other, stranger, "does not match"},
		{"a key it does not know", file, stranger, "does not know"},
		{"made for git", gitFile, gitPublic, "not for a build"},
		{"not our key at all", file, "ssh-ed25519 AAAA", "does not know"},
	}
	for _, example := range cases {
		got, err := signerAmong(example.file,
			map[string]string{"tester": example.public})
		if err == nil || !strings.Contains(err.Error(), example.want) {
			t.Errorf("%s: got %q, %v; want an error saying %q",
				example.name, got, err, example.want)
		}
	}
}

func TestSignerRefusesDamage(t *testing.T) {
	cases := []struct {
		name   string
		change func([]byte) []byte
		want   string
	}{
		{"version 2", func(raw []byte) []byte {
			binary.BigEndian.PutUint32(raw[len(magic):], 2)
			return raw
		}, "unknown version"},
		{"bytes left over", func(raw []byte) []byte {
			return append(raw, 0)
		}, "left over"},
		{"cut short", func(raw []byte) []byte {
			return raw[:len(raw)-10]
		}, "cut short"},
		{"not ssh at all", func([]byte) []byte {
			return []byte("hello")
		}, "not an ssh signature"},
	}
	for _, example := range cases {
		file, public := signed(t, "build")
		rewritten(t, file, example.change)
		got, err := signerAmong(file, map[string]string{"tester": public})
		if err == nil || !strings.Contains(err.Error(), example.want) {
			t.Errorf("%s: got %q, %v; want an error saying %q",
				example.name, got, err, example.want)
		}
	}
}

// The tests below make their own signatures, in Go. They need no
// ssh-keygen and no key anyone keeps. Each is what ssh-keygen -Y sign
// writes for the same key, file and namespace.

// sshString is data as the ssh wire format holds it: a four-byte length,
// then the bytes.
func sshString(data []byte) []byte {
	size := uint32(len(data))
	out := binary.BigEndian.AppendUint32(nil, size)
	return append(out, data...)
}

// joined is the ssh strings of parts, one after the other.
func joined(parts ...[]byte) []byte {
	var out []byte
	for _, part := range parts {
		out = append(out, sshString(part)...)
	}
	return out
}

// maker is a key a test signs with. Its seed is fixed and public: it
// signs nothing that matters.
type maker struct{ private ed25519.PrivateKey }

// makerFrom is the key whose seed is the letter repeated.
func makerFrom(letter string) maker {
	seed := bytes.Repeat([]byte(letter), ed25519.SeedSize)
	return maker{ed25519.NewKeyFromSeed(seed)}
}

// wire is the key as ssh holds it: its kind, then its bytes.
func (key maker) wire() []byte {
	public := key.private.Public().(ed25519.PublicKey)
	return joined([]byte("ssh-ed25519"), public)
}

// line is the key as an authorized_keys line holds it.
func (key maker) line() string {
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(key.wire())
}

// digest is the body hashed as hash says. Any other hash gets sha512,
// since such a signature is refused before its digest is read.
func digest(body []byte, hash string) []byte {
	if hash == "sha256" {
		sum := sha256.Sum256(body)
		return sum[:]
	}
	sum := sha512.Sum512(body)
	return sum[:]
}

// signing is what to put in a signature the key makes: the namespace and
// the hash it names, and the kind it says the signature is.
type signing struct {
	space, hash, kind string
}

// good is a signing for a build, hashed with sha512.
var good = signing{"build", "sha512", "ssh-ed25519"}

// raw is a signature of body in the layout ssh-keygen -Y sign writes,
// before it is armoured. The bytes it signs come from the code's own
// signedData. So the tests that use it check parsing and refusal, and
// not those bytes. TestSignerVerifies runs the real ssh-keygen and is the
// one test that does.
func (key maker) raw(how signing, body []byte) []byte {
	probe := signature{namespace: []byte(how.space), hash: []byte(how.hash)}
	hashed := digest(body, how.hash)
	signed := ed25519.Sign(key.private, probe.signedData(hashed))
	blob := joined([]byte(how.kind), signed)
	out := binary.BigEndian.AppendUint32([]byte(magic), version)
	fields := joined(key.wire(), []byte(how.space), nil,
		[]byte(how.hash), blob)
	return append(out, fields...)
}

// version is the version of the ssh signature format. It's defined by
// ssh's PROTOCOL.sshsig: 1.
const version = 1

// armoured is raw as the text ssh-keygen writes.
func armoured(raw []byte) []byte {
	return []byte(armourTop + "\n" + base64.StdEncoding.EncodeToString(raw) +
		"\n" + armourBottom + "\n")
}

// failer is what the helpers need of a test: a testing.T and a rapid.T
// are both one.
type failer interface {
	Helper()
	Fatal(args ...any)
}

// scratch is the directory every test makes its files in. TestMain makes
// it and removes it.
var scratch string

// failedSetup is the exit code of a test run that could not make scratch.
// It's defined: 1.
const failedSetup = 1

// TestMain makes scratch for the tests and removes it after them.
func TestMain(suite *testing.M) {
	dir, err := os.MkdirTemp("", "who-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(failedSetup)
	}
	scratch = dir
	code := suite.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// room is a directory of its own in scratch.
func room(t failer) string {
	t.Helper()
	dir, err := os.MkdirTemp(scratch, "room-")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// placed is a file holding body, with the signature raw beside it. It is
// the file's path.
func placed(t failer, body, raw []byte) string {
	t.Helper()
	file := filepath.Join(room(t), "gort")
	if err := os.WriteFile(file, body, perm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file+".sig", armoured(raw), perm); err != nil {
		t.Fatal(err)
	}
	return file
}

// perm is the mode the test files are made with. It's defined: owner
// reads and writes, everyone else reads.
const perm = 0o644

// knownTo is the one key in a set of known keys, under the name tester.
func knownTo(key maker) map[string]string {
	return map[string]string{"tester": key.line()}
}

// hashes are the two a signature may name.
var hashes = []string{"sha256", "sha512"}

// The signature verifies and names the key's owner. Any body does, and
// either hash.
func TestASignatureMadeHereVerifies(t *testing.T) {
	rapid.Check(t, checkVerifies)
}

// checkVerifies is one draw of that property.
func checkVerifies(t *rapid.T) {
	body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")
	hash := rapid.SampledFrom(hashes).Draw(t, "hash")
	key := makerFrom("a")
	how := signing{"build", hash, "ssh-ed25519"}
	file := placed(t, body, key.raw(how, body))
	if got, err := signerAmong(file, knownTo(key)); got != "tester" ||
		err != nil {
		t.Fatalf("%s: got %q, %v", hash, got, err)
	}
}

// Of two known keys, the signature names the one that made it.
func TestTheSignatureNamesItsKey(t *testing.T) {
	first, second := makerFrom("a"), makerFrom("b")
	known := map[string]string{"first": first.line(),
		"second": second.line()}
	body := []byte("a binary")
	for name, key := range map[string]maker{"first": first,
		"second": second} {
		file := placed(t, body, key.raw(good, body))
		if got, err := signerAmong(file, known); got != name || err != nil {
			t.Errorf("signed by %s: got %q, %v", name, got, err)
		}
	}
}

// Anti-test: any change to the signed file is refused. The refusal says
// it changed.
func TestAnyChangeToTheBodyIsRefused(t *testing.T) {
	rapid.Check(t, checkBodyChanged)
}

// checkBodyChanged is one draw of that property.
func checkBodyChanged(t *rapid.T) {
	body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")
	extra := rapid.Byte().Draw(t, "extra")
	other := append(bytes.Clone(body), extra)
	key := makerFrom("a")
	file := placed(t, other, key.raw(good, body))
	got, err := signerAmong(file, knownTo(key))
	if got != "" || err == nil || !strings.Contains(err.Error(),
		"does not match") {
		t.Fatalf("got %q, %v", got, err)
	}
}

const (
	// first is the first place in bytes. It's defined.
	first = 0
	// one is a place or a byte. It's defined.
	one = 1
)

// somewhereIn draws a place inside raw, named label.
func somewhereIn(t *rapid.T, raw []byte, label string) int {
	return rapid.IntRange(first, len(raw)-one).Draw(t, label)
}

// flipped is raw with the bits of the byte at place turned over.
func flipped(raw []byte, place int) []byte {
	out := bytes.Clone(raw)
	out[place] ^= flip
	return out
}

// flip turns over every bit of a byte. It's defined: all eight set.
const flip = 0xff

// Anti-test: changing any byte of a signature stops it verifying. It is
// a bounce and never a name.
func TestNoByteOfASignatureMayChange(t *testing.T) {
	rapid.Check(t, checkByteChanged)
}

// checkByteChanged is one draw of that property.
func checkByteChanged(t *rapid.T) {
	body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")
	key := makerFrom("a")
	whole := key.raw(good, body)
	place := somewhereIn(t, whole, "place")
	file := placed(t, body, flipped(whole, place))
	if got, err := signerAmong(file, knownTo(key)); got != "" || err == nil {
		t.Fatalf("byte %d changed: got %q, %v", place, got, err)
	}
}

// Anti-test: a signature cut anywhere is refused, whole or in part.
func TestNoSignatureCutShortVerifies(t *testing.T) {
	rapid.Check(t, checkCutShort)
}

// checkCutShort is one draw of that property.
func checkCutShort(t *rapid.T) {
	body := rapid.SliceOf(rapid.Byte()).Draw(t, "body")
	key := makerFrom("a")
	whole := key.raw(good, body)
	at := somewhereIn(t, whole, "at")
	file := placed(t, body, whole[:at])
	if got, err := signerAmong(file, knownTo(key)); got != "" || err == nil {
		t.Fatalf("cut at %d: got %q, %v", at, got, err)
	}
}

// refusal is a signature that is wrong in one way: what it is signed
// with, and what the refusal says.
type refusal struct {
	how  signing
	says string
}

// refusals are the refusals, by what is wrong.
func refusals() map[string]refusal {
	return map[string]refusal{
		"a hash that is neither": {signing{"build", "md5", "ssh-ed25519"},
			"neither sha256 nor sha512"},
		"made for git": {signing{"git", "sha512", "ssh-ed25519"},
			"not for a build"},
		"not an ed25519 signature": {signing{"build", "sha512", "ssh-rsa"},
			"not an ed25519 signature"},
	}
}

// Anti-test: a signature with another hash, namespace or kind is
// refused. The refusal says which.
func TestASignatureWrongInOnePartIsRefused(t *testing.T) {
	key := makerFrom("a")
	body := []byte("a binary")
	for why, refusal := range refusals() {
		file := placed(t, body, key.raw(refusal.how, body))
		got, err := signerAmong(file, knownTo(key))
		if got != "" || err == nil ||
			!strings.Contains(err.Error(), refusal.says) {
			t.Errorf("%s: got %q, %v", why, got, err)
		}
	}
}

// Anti-test: a signature by a key that is not known is refused, and so
// is one by a known key whose line has no key in it.
func TestAKeyNotKnownIsRefused(t *testing.T) {
	body := []byte("a binary")
	stranger := makerFrom("a")
	file := placed(t, body, stranger.raw(good, body))
	for why, known := range map[string]map[string]string{
		"another key":               knownTo(makerFrom("b")),
		"no keys":                   {},
		"a line with no key":        {"tester": "ssh-ed25519"},
		"a line that is not base64": {"tester": "ssh-ed25519 !!!"},
	} {
		got, err := signerAmong(file, known)
		if got != "" || err == nil ||
			!strings.Contains(err.Error(), "does not know") {
			t.Errorf("%s: got %q, %v", why, got, err)
		}
	}
}

// Anti-test: a known key that is not an ed25519 key cannot sign, even
// when the signature names that very key.
func TestAKnownKeyThatIsNotEd25519IsRefused(t *testing.T) {
	body := []byte("a binary")
	key := makerFrom("a")
	for why, wire := range map[string][]byte{
		"another kind": joined([]byte("ssh-rsa"), []byte("0123456789")),
		"a short key":  joined([]byte("ssh-ed25519"), []byte("short")),
		"not strings":  []byte("xx"),
	} {
		raw := key.raw(good, body)
		old, now := sshString(key.wire()), sshString(wire)
		raw = bytes.Replace(raw, old, now, one)
		file := placed(t, body, raw)
		known := map[string]string{"tester": "ssh-ed25519 " +
			base64.StdEncoding.EncodeToString(wire)}
		got, err := signerAmong(file, known)
		if got != "" || err == nil {
			t.Errorf("%s: got %q, %v", why, got, err)
		}
	}
}

// Anti-test: text that is not an armoured ssh signature is refused.
func TestTextThatIsNotASignatureIsRefused(t *testing.T) {
	for why, text := range map[string]string{
		"no armour":  "hello",
		"empty":      "",
		"not base64": armourTop + "\n!!!\n" + armourBottom,
		"not a signature": armourTop + "\n" +
			base64.StdEncoding.EncodeToString([]byte("hello")) + "\n" +
			armourBottom,
	} {
		file := filepath.Join(room(t), "gort")
		if err := os.WriteFile(file, []byte("a"), perm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file+".sig", []byte(text), perm); err != nil {
			t.Fatal(err)
		}
		known := knownTo(makerFrom("a"))
		got, err := signerAmong(file, known)
		if got != "" || err == nil ||
			!strings.Contains(err.Error(), "not an ssh signature") {
			t.Errorf("%s: got %q, %v", why, got, err)
		}
	}
}

// Anti-test: a signature file or a signed file that exists and cannot be
// read is an error, not nobody. Here each is a directory.
func TestAFileThatCannotBeReadIsAnError(t *testing.T) {
	key := makerFrom("a")
	body := []byte("a binary")
	dir := room(t)
	asDir := filepath.Join(dir, "gort")
	if err := os.MkdirAll(asDir+".sig", perm|dirBits); err != nil {
		t.Fatal(err)
	}
	if got, err := signerAmong(asDir, knownTo(key)); got != "" ||
		err == nil || strings.Contains(err.Error(), "does not know") {
		t.Errorf("a signature that is a directory: got %q, %v", got, err)
	}
	other := filepath.Join(dir, "second")
	if err := os.MkdirAll(other, perm|dirBits); err != nil {
		t.Fatal(err)
	}
	text := armoured(key.raw(good, body))
	if err := os.WriteFile(other+".sig", text, perm); err != nil {
		t.Fatal(err)
	}
	if got, err := signerAmong(other, knownTo(key)); got != "" ||
		err == nil {
		t.Errorf("a signed file that is a directory: got %q, %v", got, err)
	}
}

// dirBits make a directory searchable. It's defined: owner execute.
const dirBits = 0o100

// Of always names the command, and says it is unsigned exactly when no
// one signed it.
func TestOfNamesTheCommandAndWhoSignedIt(t *testing.T) {
	rapid.Check(t, checkOf)
}

// checkOf is one draw of that property.
func checkOf(t *rapid.T) {
	word := rapid.StringMatching(`[a-z0-9-]{1,8}`)
	name := word.Draw(t, "name")
	who := rapid.OneOf(rapid.Just(""), word).Draw(t, "who")
	sids := rapid.OneOf(rapid.Just(""), word).Draw(t, "sids")
	signer := rapid.OneOf(rapid.Just(""), word).Draw(t, "signer")
	got := Of(name, who, sids, signer)
	unsigned := strings.Contains(got, name+" is not signed")
	signed := strings.Contains(got, name+" was signed by "+signer)
	if unsigned == (signer != "") || signed != (signer != "") {
		t.Fatalf("%q: %q", signer, got)
	}
	if who != "" && !strings.Contains(got, "was built by "+who) {
		t.Fatalf("who %q not in %q", who, got)
	}
}

// Anti-test: with no builder Of says the build was plain go. It names no
// sessions.
func TestOfWithNoBuilderSaysNoSessions(t *testing.T) {
	rapid.Check(t, checkNoBuilder)
}

// checkNoBuilder is one draw of that property.
func checkNoBuilder(t *rapid.T) {
	sids := rapid.StringMatching(`[a-z0-9,]{0,12}`).Draw(t, "sids")
	got := Of("gort", "", sids, "")
	if !strings.Contains(got, "built by go rather than by game") ||
		strings.Contains(got, "last sessions") {
		t.Fatalf("%q: %q", sids, got)
	}
}

// Signer says the read error itself when the signature or the signed file
// cannot be read. It does not go on to complain about what it read.
func TestAnUnreadableFileIsRefusedWithTheReadError(t *testing.T) {
	key := makerFrom("a")
	dir := room(t)
	asDir := filepath.Join(dir, "gort")
	if err := os.MkdirAll(asDir+".sig", perm|dirBits); err != nil {
		t.Fatal(err)
	}
	_, err := signerAmong(asDir, knownTo(key))
	if !errors.Is(err, syscall.EISDIR) {
		t.Errorf("a signature that is a directory: got %v", err)
	}
	other := filepath.Join(dir, "second")
	if err := os.MkdirAll(other, perm|dirBits); err != nil {
		t.Fatal(err)
	}
	raw := key.raw(good, []byte("a binary"))
	if err := os.WriteFile(other+".sig", armoured(raw), perm); err != nil {
		t.Fatal(err)
	}
	_, err = signerAmong(other, knownTo(key))
	if !errors.Is(err, syscall.EISDIR) {
		t.Errorf("a signed file that is a directory: got %v", err)
	}
}

// Anti-test: a readable signed file is not a read error.
func TestAReadableFileIsNotAReadError(t *testing.T) {
	key := makerFrom("a")
	body := []byte("a binary")
	file := placed(t, body, key.raw(good, body))
	got, err := signerAmong(file, knownTo(key))
	if err != nil || got != "tester" {
		t.Errorf("got %q, %v; want tester", got, err)
	}
}

// Armour with the top line missing is refused. So is armour with the
// bottom line missing.
func TestHalfAnArmourIsRefused(t *testing.T) {
	key := makerFrom("a")
	raw := key.raw(good, []byte("a binary"))
	armour := armoured(raw)
	text := strings.TrimSpace(string(armour))
	noTop := strings.TrimPrefix(text, armourTop)
	noBottom := strings.TrimSuffix(text, armourBottom)
	neither := strings.TrimPrefix(noBottom, armourTop)
	for why, half := range map[string]string{"no top": noTop,
		"no bottom": noBottom, "neither": neither} {
		_, err := parseSignature([]byte(half))
		if err == nil || !strings.Contains(err.Error(), "not an ssh signature") {
			t.Errorf("%s: got %v", why, err)
		}
	}
}

// Anti-test: the same signature with both armour lines is read.
func TestWholeArmourIsRead(t *testing.T) {
	key := makerFrom("a")
	raw := key.raw(good, []byte("a binary"))
	sig, err := parseSignature(armoured(raw))
	if err != nil {
		t.Fatal(err)
	}
	if string(sig.namespace) != namespace {
		t.Errorf("got %q", sig.namespace)
	}
}

// Signing that ends right after its version is cut short. The version is
// known.
func TestASignatureEndingAtItsVersionIsCutShort(t *testing.T) {
	atVersion := binary.BigEndian.AppendUint32([]byte(magic), version)
	_, err := parseSignature(armoured(atVersion))
	if err == nil || !strings.Contains(err.Error(), "cut short") {
		t.Errorf("ends at its version: got %v", err)
	}
}

// Anti-test: a signature that ends inside its version has an unknown
// version. So does one whose version is not 1.
func TestASignatureWithNoUsableVersionIsRefused(t *testing.T) {
	inside := make([]byte, versionBytes-one)
	short := append([]byte(magic), inside...)
	other := binary.BigEndian.AppendUint32([]byte(magic), version+one)
	for why, raw := range map[string][]byte{
		"inside the version": short,
		"another version":    other,
	} {
		_, err := parseSignature(armoured(raw))
		if err == nil || !strings.Contains(err.Error(), "unknown version") {
			t.Errorf("%s: got %v", why, err)
		}
	}
}

// versionBytes is how long the version is in a signature. It's defined by
// ssh's PROTOCOL.sshsig: a four-byte number.
const versionBytes = 4

// Joined ssh strings read back as they were written. That holds for empty
// ones, which are only a four-byte length.
func TestFieldsOfReadsBackWhatWasJoined(t *testing.T) {
	rapid.Check(t, checkFieldsRoundTrip)
}

const (
	// maxPart is the longest string drawn. It's defined: short enough to
	// run fast, long enough to be more than a byte.
	maxPart = 6
	// maxParts is the most strings drawn. It's defined.
	maxParts = 4
	// extra is a byte left over after the strings. It's defined: zero.
	extra = 0
)

// stringsDrawn draws the parts a test joins, at least one and empty ones
// among them.
func stringsDrawn(t *rapid.T) [][]byte {
	part := rapid.SliceOfN(rapid.Byte(), first, maxPart)
	return rapid.SliceOfN(part, one, maxParts).Draw(t, "parts")
}

// checkFieldsRoundTrip is one draw of that property.
func checkFieldsRoundTrip(t *rapid.T) {
	parts := stringsDrawn(t)
	got, err := fieldsOf(joined(parts...), len(parts))
	if err != nil || len(got) != len(parts) {
		t.Fatalf("%q: got %q, %v", parts, got, err)
	}
	for at := range parts {
		if !bytes.Equal(got[at], parts[at]) {
			t.Fatalf("part %d: got %q, want %q", at, got[at], parts[at])
		}
	}
}

// Anti-test: asking for one string too many is refused. So is a byte left
// over, and so is a cut string.
func TestFieldsOfRefusesTooManyAndTooFew(t *testing.T) {
	rapid.Check(t, checkFieldsRefused)
}

// checkFieldsRefused is one draw of that property.
func checkFieldsRefused(t *rapid.T) {
	parts := stringsDrawn(t)
	whole := joined(parts...)
	if _, err := fieldsOf(whole, len(parts)+one); err == nil {
		t.Fatalf("%q: one more string than there is was read", parts)
	}
	if _, err := fieldsOf(append(whole, extra), len(parts)); err == nil {
		t.Fatalf("%q: a byte left over was read", parts)
	}
	if _, err := fieldsOf(whole[:len(whole)-one], len(parts)); err == nil {
		t.Fatalf("%q: a string cut short was read", parts)
	}
}

// Signer refuses a known key of another kind as not ed25519. That holds
// when the key has the right length and is the key that signed.
func TestAKnownKeyOfAnotherKindIsRefusedAsNotEd25519(t *testing.T) {
	body := []byte("a binary")
	key := makerFrom("a")
	public := key.private.Public().(ed25519.PublicKey)
	wire := joined([]byte("ssh-rsa"), public)
	old, now := sshString(key.wire()), sshString(wire)
	raw := bytes.Replace(key.raw(good, body), old, now, one)
	file := placed(t, body, raw)
	known := map[string]string{"tester": "ssh-ed25519 " +
		base64.StdEncoding.EncodeToString(wire)}
	got, err := signerAmong(file, known)
	if got != "" || err == nil || !strings.Contains(err.Error(), "not ed25519") {
		t.Errorf("got %q, %v", got, err)
	}
}

// Anti-test: the same key under its own kind is accepted.
func TestTheSameKeyUnderItsOwnKindIsAccepted(t *testing.T) {
	body := []byte("a binary")
	key := makerFrom("a")
	file := placed(t, body, key.raw(good, body))
	got, err := signerAmong(file, knownTo(key))
	if got != "tester" || err != nil {
		t.Errorf("got %q, %v", got, err)
	}
}

// shortLine is a key line whose key has too few bytes to be ed25519.
func (maker) shortLine() string {
	wire := joined([]byte("ssh-ed25519"), []byte("short"))
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(wire)
}

// vouching is the signers a test vouches for: the test key "a" as ada.
func vouching(t *testing.T) Signers {
	t.Helper()
	var vouched Signers
	if err := vouched.Add("ada", makerFrom("a").line()); err != nil {
		t.Fatal(err)
	}
	return vouched
}

// A configured signer is vouched for exactly as a known key was: SignedBy
// names it.
func TestAConfiguredSignerIsNamedBySignedBy(t *testing.T) {
	body := []byte("a binary")
	file := placed(t, body, makerFrom("a").raw(good, body))
	got, err := SignedBy(file, vouching(t))
	if got != "ada" || err != nil {
		t.Errorf("got %q, %v; want ada", got, err)
	}
}

// Anti-test: with no signer set, a signature is not vouched for. Signer
// names nobody, and says which key it does not know.
func TestWithNoSignerNothingIsVouchedFor(t *testing.T) {
	body := []byte("a binary")
	file := placed(t, body, makerFrom("a").raw(good, body))
	var nobody Signers
	got, err := SignedBy(file, nobody)
	if got != "" || err == nil ||
		!strings.Contains(err.Error(), "does not know") {
		t.Errorf("got %q, %v", got, err)
	}
	if nobody.Any() {
		t.Error("no signers was said to have one")
	}
}

// Anti-test: no signature is nobody and no error, whoever is vouched for.
func TestNoSignatureIsNobodyWhoIsVouchedFor(t *testing.T) {
	bare := room(t) + "/gort"
	for _, vouched := range []Signers{{}, vouching(t)} {
		if got, err := SignedBy(bare, vouched); got != "" || err != nil {
			t.Errorf("got %q, %v", got, err)
		}
	}
}

// Anti-test: a path that is empty names no file, so no signature is read
// from the working directory and no one signed it.
func TestAnEmptyPathHasNoSignature(t *testing.T) {
	if got, err := SignedBy("", vouching(t)); got != "" || err != nil {
		t.Errorf("got %q, %v", got, err)
	}
}

// Signers say they have one once Add has taken one.
func TestSignersWithOneSayAny(t *testing.T) {
	if !vouching(t).Any() {
		t.Error("a signer was added and Any is false")
	}
}

// Anti-test: Add refuses a line that is not an ed25519 key, and the
// refusal names the signer.
func TestAddRefusesALineThatIsNotAKey(t *testing.T) {
	for why, line := range map[string]string{
		"empty":        "",
		"no key":       "ssh-ed25519",
		"another kind": "ssh-rsa " + makerFrom("a").line()[len("ssh-ed25519 "):],
		"not base64":   "ssh-ed25519 !!!",
		"not a key":    "ssh-ed25519 AAAA",
		"a short key":  "ssh-ed25519 " + shortKey(),
	} {
		var vouched Signers
		err := vouched.Add("ada", line)
		if err == nil || !strings.Contains(err.Error(), "signer ada") {
			t.Errorf("%s: %v", why, err)
		}
		if vouched.Any() {
			t.Errorf("%s: a refused line was kept", why)
		}
	}
}

// shortKey is an ed25519 key line's data with too few bytes in it.
func shortKey() string {
	return strings.TrimPrefix(
		(maker{}).shortLine(), "ssh-ed25519 ")
}

// A signer line is read from its name and its tail, and keeps the line it
// was on.
func TestASignerIsReadFromItsLine(t *testing.T) {
	key := makerFrom("a").line()
	got, err := readSigners(t, "# the signers\n\nsigner ada :: "+key+"\n")
	if err != nil {
		t.Fatal(err)
	}
	want := Signer{Name: "ada", Key: key, At: signerAt}
	if len(got.Signers) != 1 || got.Signers[0] != want {
		t.Errorf("read %+v, wanted %+v", got.Signers, want)
	}
}

// signerAt is the line a test's signer is on. It's defined.
const signerAt = 3

// signerFile is a config of signers, as pzm's own has them.
type signerFile struct {
	Signers []Signer `config:"signer,1,tail" help:"a person whose signature vouches"`
}

// readSigners is the text read as a config of signers.
func readSigners(t *testing.T, text string) (*signerFile, error) {
	t.Helper()
	schema, err := config.NewSchema[signerFile]()
	if err != nil {
		t.Fatal(err)
	}
	var got signerFile
	if err := schema.Read("config", []byte(text), &got); err != nil {
		return nil, err
	}
	return &got, nil
}

// Anti-test: a signer whose key is not an ed25519 key is refused, naming
// the line and why, and keeps nothing.
func TestASignerWithABadKeyIsRefusedAndKeepsNothing(t *testing.T) {
	for why, key := range map[string]string{
		"no key":      "ssh-ed25519",
		"not base64":  "ssh-ed25519 !!!",
		"not a key":   "ssh-ed25519 AAAA",
		"a short key": "ssh-ed25519 " + shortKey(),
	} {
		got, err := readSigners(t, "signer ada :: "+key+"\n")
		if err == nil || got != nil ||
			!strings.Contains(err.Error(), "config:1: signer: ") {
			t.Errorf("%s: got %+v, %v", why, got, err)
		}
	}
}

// Anti-test: a name can be vouched for once.
func TestAddRefusesANameTwice(t *testing.T) {
	vouched := vouching(t)
	err := vouched.Add("ada", makerFrom("b").line())
	if err == nil || !strings.Contains(err.Error(), "signer ada") ||
		!strings.Contains(err.Error(), "twice") {
		t.Errorf("got %v", err)
	}
}
