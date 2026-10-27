package pzm_test

import (
	"strings"
	"testing"
)

// Anti-test: a failing verb says why on one line, and does not print its
// usage after it. Nothing reaches the output.
func TestAFailingVerbPrintsNoUsage(t *testing.T) {
	tool, _ := built()
	code, out, errs := run(tool, "baz")
	if code != failedCode || out != "" || errs != "demo baz: boom\n" {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}

// Anti-test: --age is a switch, so a value given to it is refused and
// the tool says nothing of its age.
func TestASwitchGivenAValueIsRefusedForAge(t *testing.T) {
	tool, _ := built()
	code, out, errs := run(tool, "--age=now")
	if code != refusedCode || out != "" || !strings.Contains(errs, "age") {
		t.Errorf("code %d, out %q, errs %q", code, out, errs)
	}
}
