package pzm_test

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"testing"
)

// failedCode is the exit code of a verb that ran and failed. It is
// defined by pzm.Tool.RunWith: 1.
const failedCode = 1

// runBroken runs the test tool with args and a pipe nobody reads for
// output: the other end is closed, so every write to it fails. It gives
// the exit code and the error stream.
func runBroken(t *testing.T, args []string) (int, string) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	var errs bytes.Buffer
	code := helped().RunWith(args, strings.NewReader(""), writer, &errs)
	return code, errs.String()
}

// helpAsks are the ways to ask for help from the test tool.
var helpAsks = [][]string{{}, {"--help"}, {"-h"}, {"help"},
	{"foo-bar", "--help"}, {"foo-bar", "-h"}, {"help", "foo-bar"},
	{"age", "--help"}, {"help", "who"}, {"--agent-help"}}

// Help that cannot be written is not a success. Nothing was said, so
// the exit is 2 and the write's error is on the error stream. That holds
// for the tool's help, a verb's and the agent's.
func TestHelpThatCannotBeWrittenIsRefused(t *testing.T) {
	for _, asked := range helpAsks {
		code, errs := runBroken(t, asked)
		if code != refusedCode || !strings.Contains(errs,
			syscall.EPIPE.Error()) {
			t.Errorf("%v: code %d, errs %q", asked, code, errs)
		}
	}
}

// Anti-test: the same asks written to a writer that works exit 0, with
// help and nothing on the error stream.
func TestHelpThatCanBeWrittenStillSucceeds(t *testing.T) {
	for _, asked := range helpAsks {
		code, out, errs := run(helped(), asked...)
		if code != ranCode || out == "" || errs != "" {
			t.Errorf("%v: code %d, out %q, errs %q", asked, code, out,
				errs)
		}
	}
}

// Anti-test: a verb that ran and could not write its answer is a verb
// that failed, exit 1. So is one of pzm's own verbs.
func TestAVerbThatCannotBeWrittenStillFails(t *testing.T) {
	for _, asked := range [][]string{{"foo-bar", "x"}, {"age"}} {
		code, errs := runBroken(t, asked)
		if code != failedCode || !strings.Contains(errs,
			syscall.EPIPE.Error()) {
			t.Errorf("%v: code %d, errs %q", asked, code, errs)
		}
	}
}
