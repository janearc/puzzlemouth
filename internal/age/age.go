// Package age answers --age: which commit a binary was built from, and how
// long ago that commit was made.
//
// game build stamps two variables into a command's main package with
// -ldflags: build, the short commit, and built, the commit's time. The
// command declares them and passes them here. Any binary built with plain
// go build has neither, and the answer says so.
package age

import (
	"fmt"
	"time"
)

// Of is the line a command prints for --age. name is the command, build
// and built are the two stamped variables, and now is the clock to measure
// the commit's age against.
func Of(name, build, built string, now time.Time) string {
	if build == "" || build == "dev" {
		return name + ", built by go rather than by game: " +
			"no commit and no commit time in it"
	}
	if built == "" {
		return fmt.Sprintf("%s %s, with no commit time in it",
			name, build)
	}
	when, err := time.Parse(time.RFC3339, built)
	if err != nil {
		return fmt.Sprintf("%s %s, committed at %s", name, build, built)
	}
	return fmt.Sprintf("%s %s, committed %s ago",
		name, build, now.Sub(when).Round(time.Second))
}
