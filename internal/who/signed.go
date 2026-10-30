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
	"strings"

	"github.com/janearc/puzzlemouth/config"
)

// namespace is the purpose a build signature is made for, so a signature
// made for git cannot pass as one for a build. Sign with:
//
//	ssh-keygen -Y sign -n build -f KEY bin/NAME
const namespace = "build"

// Signers are the keys whose signatures Signer accepts, by name. The zero
// value vouches for nobody. A key is added from the line an authorized_keys
// file would hold.
type Signers struct{ keys map[string]string }

// Signer is one line of pzm's config, "signer NAME :: KEY": a person
// whose signature on a build --who vouches for. Config fills it, and
// Signers.Add vouches for it. Its fields are plain, as it is only read.
type Signer struct {
	// Name is who the key belongs to.
	Name string
	// Key is the line an authorized_keys file would hold.
	Key string
	// At is the line of the config it was read from.
	At int
}

// SetLine reads a signer from the line of the config: one word, the
// name, and the key as the tail. It refuses a key that is not an
// ed25519 public key.
func (signer *Signer) SetLine(line config.Line) error {
	if err := keyRefusal(line.Tail); err != nil {
		return err
	}
	signer.Name, signer.Key, signer.At = line.Words[nameWord], line.Tail,
		line.At
	return nil
}

// nameWord is the place of the name among a signer line's words. It's
// defined: a signer line takes one.
const nameWord = 0

// keyRefusal is why a line is not an ed25519 public key, or nothing.
func keyRefusal(line string) error {
	parts := strings.Fields(line)
	if len(parts) < pairFields || parts[kindField] != "ssh-ed25519" {
		return fmt.Errorf("%q is not an ssh-ed25519 key line", line)
	}
	wire, err := base64.StdEncoding.DecodeString(parts[dataField])
	if err != nil {
		return errors.New("the key is not base64")
	}
	kind, key, err := pair(wire)
	if err != nil || kind != "ssh-ed25519" ||
		len(key) != ed25519.PublicKeySize {
		return errors.New("the key is not an ed25519 public key")
	}
	return nil
}

// Add vouches for the key on line under name. It refuses a name already
// vouched for, and a line that is not an ed25519 public key; the refusal
// names the signer.
func (vouched *Signers) Add(name, line string) error {
	if _, twice := vouched.keys[name]; twice {
		return fmt.Errorf("signer %s is set twice", name)
	}
	if err := keyRefusal(line); err != nil {
		return fmt.Errorf("signer %s: %w", name, err)
	}
	if vouched.keys == nil {
		vouched.keys = map[string]string{}
	}
	vouched.keys[name] = line
	return nil
}

// Any is whether anyone is vouched for.
func (vouched Signers) Any() bool { return len(vouched.keys) > nothing }

// SignedBy says who signed the file at path. It reads the signature
// beside it in path+".sig". No signature is "" and no error.
//
// Any signature that does not verify is an error. So is one that a key
// outside vouched made, and so is any signature when nobody is vouched
// for.
//
// A path that is empty names no file, so no one signed it.
func SignedBy(path string, vouched Signers) (string, error) {
	if path == "" {
		return "", nil
	}
	return signerAmong(path, vouched.keys)
}

// signerAmong is SignedBy with the known keys passed in, so a test can use
// keys of its own.
func signerAmong(path string, known map[string]string) (string, error) {
	armour, err := os.ReadFile(path + ".sig")
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	sig, err := parseSignature(armour)
	if err != nil {
		return "", fmt.Errorf("%s.sig: %w", path, err)
	}
	if string(sig.namespace) != namespace {
		return "", fmt.Errorf("%s.sig was made for %q, not for a build",
			path, sig.namespace)
	}
	name, key, err := whoseKey(sig.key, known)
	if err != nil {
		return "", fmt.Errorf("%s.sig: %w", path, err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	digest, err := digestOf(body, string(sig.hash))
	if err != nil {
		return "", fmt.Errorf("%s.sig: %w", path, err)
	}
	if !ed25519.Verify(key, sig.signedData(digest), sig.blob) {
		return "", fmt.Errorf("%s.sig does not match %s: the file "+
			"changed after it was signed, or the signature is "+
			"damaged", path, path)
	}
	return name, nil
}

// signature is an ssh signature, the format ssh-keygen -Y sign writes.
type signature struct {
	key, namespace, reserved, hash, blob []byte
}

const (
	armourTop    = "-----BEGIN SSH SIGNATURE-----"
	armourBottom = "-----END SSH SIGNATURE-----"
	magic        = "SSHSIG"
)

// The layout of an ssh signature is defined by the sshsig format.
const (
	// lengthBytes is the size of the length before each string. It's
	// defined.
	lengthBytes = 4
	// signatureVersion is the one signature version there is. It's defined.
	signatureVersion = 1
	// signatureFields is how many strings follow the version. It's defined.
	signatureFields = 5
	// keyField is the place of the public key among them. It's defined.
	keyField = 0
	// namespaceField is the place of the namespace. It's defined.
	namespaceField = 1
	// reservedField is the place of the reserved string. It's defined.
	reservedField = 2
	// hashField is the place of the hash's name. It's defined.
	hashField = 3
	// blobField is the place of the signature itself. It's defined.
	blobField = 4
	// pairFields is how many strings make a key or a signature. It's
	// defined.
	pairFields = 2
	// kindField is the place of the kind in a pair. It's defined.
	kindField = 0
	// dataField is the place of the bytes in a pair. It's defined.
	dataField = 1
	// nothing is a count of nothing. It's defined.
	nothing = 0
)

// parseSignature reads the armoured text ssh-keygen writes into its parts.
func parseSignature(armour []byte) (*signature, error) {
	body := strings.TrimSpace(string(armour))
	if !strings.HasPrefix(body, armourTop) ||
		!strings.HasSuffix(body, armourBottom) {
		return nil, errors.New("not an ssh signature")
	}
	body = strings.TrimSuffix(strings.TrimPrefix(body, armourTop),
		armourBottom)
	joined := strings.Join(strings.Fields(body), "")
	raw, err := base64.StdEncoding.DecodeString(joined)
	if err != nil || !bytes.HasPrefix(raw, []byte(magic)) {
		return nil, errors.New("not an ssh signature")
	}
	rest := raw[len(magic):]
	if len(rest) < lengthBytes ||
		binary.BigEndian.Uint32(rest) != signatureVersion {
		return nil, errors.New("an ssh signature of an " +
			"unknown version")
	}
	fields, err := fieldsOf(rest[lengthBytes:], signatureFields)
	if err != nil {
		return nil, err
	}
	kind, blob, err := pair(fields[blobField])
	if err != nil || kind != "ssh-ed25519" {
		return nil, errors.New("not an ed25519 signature")
	}
	return &signature{key: fields[keyField],
		namespace: fields[namespaceField],
		reserved:  fields[reservedField], hash: fields[hashField],
		blob: blob}, nil
}

// signedData is what the key actually signed. It is the magic then the
// namespace then the reserved field then the hash's name then the file's
// digest.
func (sig signature) signedData(digest []byte) []byte {
	out := []byte(magic)
	for _, part := range [][]byte{sig.namespace, sig.reserved, sig.hash,
		digest} {
		size := uint32(len(part))
		out = binary.BigEndian.AppendUint32(out, size)
		out = append(out, part...)
	}
	return out
}

// digestOf hashes the signed file the way the signature says it was
// hashed.
func digestOf(body []byte, hash string) ([]byte, error) {
	switch hash {
	case "sha512":
		sum := sha512.Sum512(body)
		return sum[:], nil
	case "sha256":
		sum := sha256.Sum256(body)
		return sum[:], nil
	}
	return nil, fmt.Errorf("signed with %q, which is neither sha256 "+
		"nor sha512", hash)
}

// whoseKey finds the name in known whose key made the signature, and
// returns that key ready to verify with.
func whoseKey(wire []byte, known map[string]string) (string,
	ed25519.PublicKey, error) {
	for name, line := range known {
		parts := strings.Fields(line)
		if len(parts) < pairFields {
			continue
		}
		ours, err := base64.StdEncoding.DecodeString(parts[dataField])
		if err != nil || !bytes.Equal(ours, wire) {
			continue
		}
		kind, key, err := pair(wire)
		if err != nil || kind != "ssh-ed25519" ||
			len(key) != ed25519.PublicKeySize {
			return "", nil, errors.New(
				"signed by a key that is not ed25519")
		}
		return name, ed25519.PublicKey(key), nil
	}
	return "", nil, errors.New("signed by a key puzzlemouth does not know")
}

// pair reads the two ssh strings that make a key or a signature. They
// are its kind, such as "ssh-ed25519", and its bytes.
func pair(wire []byte) (string, []byte, error) {
	both, err := fieldsOf(wire, pairFields)
	if err != nil {
		return "", nil, err
	}
	return string(both[kindField]), both[dataField], nil
}

// fieldsOf reads count ssh strings. Each is a four-byte length then that
// many bytes. It refuses anything left over after them.
func fieldsOf(data []byte, count int) ([][]byte, error) {
	var out [][]byte
	for range count {
		if len(data) < lengthBytes {
			return nil, errors.New("an ssh signature cut short")
		}
		size := binary.BigEndian.Uint32(data)
		data = data[lengthBytes:]
		if uint64(len(data)) < uint64(size) {
			return nil, errors.New("an ssh signature cut short")
		}
		out = append(out, data[:size])
		data = data[size:]
	}
	if len(data) != nothing {
		return nil, errors.New("an ssh signature with bytes left over")
	}
	return out, nil
}
