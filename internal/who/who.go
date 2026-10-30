// Package who answers --who: who built a binary, and which sessions last
// committed to the code it was built from.
//
// game build stamps two variables into a command's main package, beside
// build and built. who is the session that ran the build and the machine
// it ran on, such as "sid/1a2b3c4d on box", or "no session on box".
//
// sids is the last three different session lines in the commits behind
// the build, joined with commas. Any binary built with plain go build has
// neither.
//
// The session line traces a build; it does not prove one, since anything
// can set or unset an environment variable. Proof is a signature made
// with a key that needs Touch ID, kept beside the binary as NAME.sig.
//
// SignedBy checks that signature from inside the binary, as a convenience:
// a binary someone has tampered with can print whatever it likes. The
// check that counts runs outside it, with a file of lines, each a name
// and a public key, given to -f:
//
//	ssh-keygen -Y verify -f SIGNERS -I NAME -n build \
//		-s bin/gort.sig < bin/gort
package who

import "strings"

// Of is what a command prints for --who. name is the command, who and
// sids are the two stamped variables, and signer is the name SignedBy
// returned, or "" for nobody.
func Of(name, who, sids, signer string) string {
	var lines []string
	if signer == "" {
		lines = append(lines, name+" is not signed")
	} else {
		checked := name + " was signed by " + signer +
			", as " + name + " itself checked;"
		proof := "for proof, check " + name + ".sig with " +
			"ssh-keygen -Y verify"
		lines = append(lines, checked, proof)
	}
	if who == "" {
		lines = append(lines, name+" was built by go rather "+
			"than by game: no builder and no sessions in it")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, name+" was built by "+who)
	if sids == "" {
		lines = append(lines, "no session signed the commits behind it")
	} else {
		lines = append(lines, "the last sessions to commit: "+
			strings.ReplaceAll(sids, ",", ", "))
	}
	return strings.Join(lines, "\n")
}
