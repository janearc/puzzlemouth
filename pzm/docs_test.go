package pzm_test

import (
	"os"
	"strings"
	"testing"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// samples are the Go blocks in a Markdown file, each as its lines.
func samples(t *testing.T, path string) [][]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var found [][]string
	var block []string
	inside := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case line == "```go":
			inside, block = true, nil
		case inside && line == "```":
			inside = false
			found = append(found, block)
		case inside:
			block = append(block, line)
		}
	}
	return found
}

// body is the lines of the function name in source, from its opening
// line to the closing brace at the start of a line, each trimmed.
func body(source, name string) []string {
	var lines []string
	inside := false
	for _, line := range strings.Split(source, "\n") {
		switch {
		case strings.HasPrefix(line, "func "+name+"() {"):
			inside = true
		case inside && line == "}":
			return lines
		case inside:
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return nil
}

// holds is whether the sample's lines are an unbroken run of the body's
// lines, in order. Blank lines, and main's os.Exit, which no example can
// hold, are not part of the run.
func holds(sample, lines []string) bool {
	var run []string
	for _, line := range sample {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "os.Exit(") {
			run = append(run, line)
		}
	}
	if len(run) == none {
		return false
	}
	for start := 0; start+len(run) <= len(lines); start++ {
		if strings.Join(lines[start:start+len(run)], "\n") ==
			strings.Join(run, "\n") {
			return true
		}
	}
	return false
}

// holds finds a sample only whole and in order: one scrambled, cut short
// of a line, or broken by a line between is not held.
func TestHoldsTakesOnlyTheWholeSample(t *testing.T) {
	lines := []string{"a := 1", "b := 2", "c := 3", "d := 4"}
	for _, each := range []struct {
		sample []string
		want   bool
	}{
		{[]string{"b := 2", "\tc := 3"}, true},
		{[]string{"b := 2", "", "c := 3", "os.Exit(0)"}, true},
		{[]string{"c := 3", "b := 2"}, false},
		{[]string{"b := 2", "d := 4"}, false},
		{[]string{"b := 2", "c := 3", "e := 5"}, false},
		{[]string{"os.Exit(0)"}, false},
	} {
		if got := holds(each.sample, lines); got != each.want {
			t.Errorf("%q: %v, want %v", each.sample, got, each.want)
		}
	}
}

// refusals are the calls doc/flags.md shows under "what is refused", each
// with what the page says the tool answers.
func refusals(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile("../doc/flags.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, _ := strings.Cut(string(raw), "## what is refused")
	section, _, _ = strings.Cut(section, "\n## ")
	said := map[string]string{}
	call := ""
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimPrefix(line, "    ")
		switch {
		case strings.HasPrefix(line, "hi: "):
			said[call] = line
		case strings.HasPrefix(line, "hi "):
			call = line
		}
	}
	return said
}

// Each refusal on doc/flags.md is what the page's sample tool says, word
// for word, and nothing runs.
func TestTheRefusalsOnThePageAreTheTools(t *testing.T) {
	said := refusals(t)
	if len(said) < 5 {
		t.Fatalf("doc/flags.md shows %d refusals", len(said))
	}
	for call, want := range said {
		tool := pzm.New("hi", "says hello", "", "", "", "")
		tool.Flags(pzm.Value("lang", verb.NoShort, "the language to say it in"))
		tool.Verb("hello", verb.NoShort, "says hello", "# hello", "Example_hello", readsOnly,
			hello, pzm.Switch("loud", 'l', "says it louder"),
			pzm.Value("name", 'n', "who to say hello to"),
			pzm.List("to", verb.NoShort, "everyone to say hello to"))
		var out, errs strings.Builder
		code := tool.Run(strings.Fields(call)[1:], &out, &errs)
		if code != exitRefused || out.String() != "" || errs.String() != want+"\n" {
			t.Errorf("%s: code %d, out %q, errs %q, page %q", call, code,
				out.String(), errs.String(), want)
		}
	}
}

// Each page's Go sample runs. It is an unbroken run of the lines of the
// example the page names, in their order. So the docs cannot drift from
// what runs. The pages are the README and three in doc/.
func TestTheDocsSamplesRun(t *testing.T) {
	var raw []byte
	for _, file := range []string{"flag_example_test.go",
		"dryrun_example_test.go", "bare_example_test.go"} {
		each, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, each...)
	}
	for path, example := range map[string]string{
		"../README.md":    "Example_hello",
		"../doc/flags.md": "Example_helloFlags",
		"../doc/facts.md": "Example_tidy",
		"../doc/bare.md":  "Example_look",
	} {
		lines := body(string(raw), example)
		if lines == nil {
			t.Errorf("%s: there is no %s", path, example)
			continue
		}
		blocks := samples(t, path)
		if len(blocks) != one {
			t.Errorf("%s has %d Go samples, not one", path, len(blocks))
		}
		for _, block := range blocks {
			if !holds(block, lines) {
				t.Errorf("%s: %s does not hold its sample whole and "+
					"in order", path, example)
			}
		}
	}
	// facts.md's sample is about its last line, the dry run. holds lets
	// a run of lines stop short, so the end is held here
	blocks := samples(t, "../doc/facts.md")
	if len(blocks) == none || !strings.Contains(lastLine(blocks[0]),
		`"--dry-run"`) {
		t.Errorf("doc/facts.md's sample does not end on its dry run")
	}
}

// lastLine is a sample's last line with anything on it.
func lastLine(sample []string) string {
	for at := len(sample) - 1; at >= 0; at-- {
		if strings.TrimSpace(sample[at]) != "" {
			return sample[at]
		}
	}
	return ""
}
