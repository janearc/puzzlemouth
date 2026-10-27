package age_test

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/internal/age"
	"pgregory.net/rapid"
)

// now is the clock the properties measure against: a fixed day, so a
// failure names a date a reader can follow.
var now, _ = time.Parse(time.RFC3339, "2026-10-09T23:00:00Z")

// longest is the furthest back a drawn commit may be, a hundred years of
// hours.
var longest, _ = time.ParseDuration("876000h")

// noSeconds is a commit made this second. It's defined.
const noSeconds = 0

// ago draws how long before now a commit was made.
func ago(t *rapid.T) time.Duration {
	most := int64(longest / time.Second)
	seconds := rapid.Int64Range(noSeconds, most).Draw(t, "seconds")
	return time.Duration(seconds) * time.Second
}

// commits are the short commits a build may be stamped with.
var commits = []string{"284aab1", "284aab1-dirty", "0000000", "abc"}

// Of says how old a commit is to the second. It names the command and
// the short commit.
func TestAgeIsTheTimeBetweenTheCommitAndNow(t *testing.T) {
	rapid.Check(t, checkAge)
}

// checkAge is one draw of that property.
func checkAge(t *rapid.T) {
	back := ago(t)
	commit := rapid.SampledFrom(commits).Draw(t, "commit")
	built := now.Add(-back).Format(time.RFC3339)
	want := "gort " + commit + ", committed " + back.String() + " ago"
	if got := age.Of("gort", commit, built, now); got != want {
		t.Fatalf("%s: got %q, want %q", built, got, want)
	}
}

// Anti-test: with no commit or the commit dev, Of says go built it and
// names no commit.
func TestAnUnstampedBuildNamesNoCommit(t *testing.T) {
	rapid.Check(t, checkUnstamped)
}

// checkUnstamped is one draw of that property.
func checkUnstamped(t *rapid.T) {
	build := rapid.SampledFrom([]string{"", "dev"}).Draw(t, "build")
	built := now.Add(-ago(t)).Format(time.RFC3339)
	got := age.Of("gort", build, built, now)
	if !strings.Contains(got, "built by go rather than by game") ||
		strings.Contains(got, "ago") || strings.Contains(got, built) {
		t.Fatalf("%q, %q: %q", build, built, got)
	}
}

// Anti-test: a commit time that is not a time is given back as it was
// written, with no age made up for it.
func TestACommitTimeThatIsNoTimeGetsNoAge(t *testing.T) {
	rapid.Check(t, checkNoTime)
}

// checkNoTime is one draw of that property.
func checkNoTime(t *rapid.T) {
	built := rapid.StringMatching(`[a-z ]{1,12}`).Draw(t, "built")
	want := "gort 284aab1, committed at " + built
	if got := age.Of("gort", "284aab1", built, now); got != want {
		t.Fatalf("%q: got %q", built, got)
	}
}
