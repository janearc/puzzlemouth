package age_test

import (
	"testing"
	"time"

	"github.com/janearc/puzzlemouth/internal/age"
)

func TestOf(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2026-10-09T23:00:00Z")
	cases := []struct {
		build, built, want string
	}{
		{"dev", "", "gort, built by go rather than by game: " +
			"no commit and no commit time in it"},
		{"", "", "gort, built by go rather than by game: " +
			"no commit and no commit time in it"},
		{"284aab1", "", "gort 284aab1, with no commit time in it"},
		{"284aab1", "yesterday", "gort 284aab1, committed at yesterday"},
		{"284aab1-dirty", "2026-10-09T21:30:00Z",
			"gort 284aab1-dirty, committed 1h30m0s ago"},
	}
	for _, example := range cases {
		got := age.Of("gort", example.build, example.built, now)
		if got != example.want {
			t.Errorf("Of(%q, %q) = %q, want %q",
				example.build, example.built, got, example.want)
		}
	}
}
