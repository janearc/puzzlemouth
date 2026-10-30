package pzm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
)

// stamped is a tool as game would build it: its commit and that commit's
// time, the session and machine that built it, and the last sessions
// behind it.
func stamped() *pzm.Tool {
	return pzm.New("demo", "a tool for testing pzm", "abc1234",
		"2026-10-14T10:00:00Z", "sid/1a2b3c4d on box",
		"sid/aaaaaaaa,sid/bbbbbbbb")
}

// --age, and age, say which commit the tool is and how long ago it was
// made.
func TestAgeSaysTheCommit(t *testing.T) {
	for _, asked := range []string{"--age", "age"} {
		code, out, errs := run(stamped(), asked)
		if code != exitOK || !strings.HasPrefix(out, "demo abc1234, committed ") ||
			!strings.HasSuffix(out, " ago\n") {
			t.Errorf("%s: code %d, errs %q, out %q", asked, code, errs, out)
		}
	}
}

// --who, and who, say who built the tool and the sessions behind it.
func TestWhoSaysTheBuilder(t *testing.T) {
	want := "demo is not signed\ndemo was built by sid/1a2b3c4d on box\n" +
		"the last sessions to commit: sid/aaaaaaaa, sid/bbbbbbbb\n" +
		"no signer is set: no key is vouched for; add a signer line to " +
		"/pzm-tests-have-no-home/.config/pzm/config\n"
	for _, asked := range []string{"--who", "who"} {
		code, out, errs := run(stamped(), asked)
		if code != exitOK || out != want {
			t.Errorf("%s: code %d, errs %q, out %q", asked, code, errs, out)
		}
	}
}

// A tool built by plain go has no stamps, and says so.
func TestAnUnstampedToolSaysSo(t *testing.T) {
	tool, _ := built()
	if _, out, _ := run(tool, "--age"); out != "demo, built by go rather "+
		"than by game: no commit and no commit time in it\n" {
		t.Errorf("--age said %q", out)
	}
	if _, out, _ := run(tool, "--who"); !strings.Contains(out, "demo was "+
		"built by go rather than by game: no builder and no sessions in it\n") {
		t.Errorf("--who said %q", out)
	}
}

// age and who are one at a time, like any verb.
func TestAgeAndWhoAreVerbs(t *testing.T) {
	if code, _, _ := run(stamped(), "--age", "--who"); code != exitRefused {
		t.Errorf("--age --who: code %d", code)
	}
}

// A tool is made with its name, its line of help, and the four stamps
// game builds into it; a tool built by go passes them empty.
func ExampleNew() {
	tool := pzm.New("demo", "a tool for testing pzm", "", "", "", "")
	tool.Run([]string{"--age"}, os.Stdout, os.Stderr)
	// Output: demo, built by go rather than by game: no commit and no commit time in it
}
