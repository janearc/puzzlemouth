package verb_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/janearc/puzzlemouth/verb"
)

// Check and Shown say a bad build constraint as the file's own error,
// led by its name. They do not say the file is left out of go test.
func TestABadConstraintIsSaidAsBadNotAsLeftOut(t *testing.T) {
	tests := fstest.MapFS{"bad_test.go": constrained("&& ||")}
	err := verb.Check(tests, declared(t))
	if err == nil || !strings.HasPrefix(err.Error(), "bad_test.go: ") ||
		strings.Contains(err.Error(), "leaves out") {
		t.Errorf("Check said %v", err)
	}
	_, err = verb.Shown(tests, declared(t))
	if err == nil || !strings.HasPrefix(err.Error(), "bad_test.go: ") ||
		strings.Contains(err.Error(), "leaves out") {
		t.Errorf("Shown said %v", err)
	}
}

// Anti-test: a constraint that leaves the file out is said as left out.
// It is not said as bad.
func TestAConstraintThatLeavesTheFileOutIsSaidAsLeftOut(t *testing.T) {
	tests := fstest.MapFS{"bad_test.go": constrained("ignore")}
	err := verb.Check(tests, declared(t))
	if err == nil || !strings.Contains(err.Error(), "leaves out") ||
		strings.HasPrefix(err.Error(), "bad_test.go: ") {
		t.Errorf("Check said %v", err)
	}
}
