package pzm_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/janearc/puzzlemouth/pzm"
	"github.com/janearc/puzzlemouth/verb"
)

// these are the examples the README, doc/flags.md and visualdocs show.
// each is a test, so what it prints is held to what it says, and the
// samples in the docs are made of their lines (TestTheDocsSamplesRun).

// sayHello says hello.
func sayHello(call pzm.Call) error {
	_, err := fmt.Fprintln(call.Out(), "hello")
	return err
}

// hello is the verb the README and doc/flags.md declare: it says hello,
// louder with --loud.
func hello(call pzm.Call) error {
	if call.On("loud") {
		_, err := fmt.Fprintln(call.Out(), "HELLO")
		return err
	}
	return sayHello(call)
}

// The README's sample: a tool and one verb with its facts and its flag,
// then a run. Each verb's example is named Example_ and the verb, since
// go vet refuses an example named for an identifier that is not there.
func Example_hello() {
	build, built, who, sids := "", "", "", ""
	page := verb.Page("# hello\n\nsays hello.")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool := pzm.New("hi", "says hello", pzm.Build(build), pzm.Built(built),
		pzm.Who(who), pzm.Sessions(sids))
	tool.Verb("hello", verb.NoShort, "says hello", page, "Example_hello", facts, hello,
		pzm.Switch("loud", 'l', "says it louder"))
	tool.Run([]string{"hello", "--loud"}, os.Stdout, os.Stderr)
	// Output: HELLO
}

// doc/flags.md's sample: the tool's own flag, and a verb with one flag
// of each kind.
func Example_helloFlags() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	page := verb.Page("# hello\n\nsays hello.")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Flags(pzm.Value("lang", verb.NoShort, "the language to say it in"))
	tool.Verb("hello", verb.NoShort, "says hello", page, "Example_hello", facts, hello,
		pzm.Switch("loud", 'l', "says it louder"),
		pzm.Value("name", 'n', "who to say hello to"),
		pzm.List("to", verb.NoShort, "everyone to say hello to"))
	tool.Run([]string{"--lang", "fr", "hello", "-l", "--to", "ada,bo"},
		os.Stdout, os.Stderr)
	// Output: HELLO
}

// A switch is on when it is given, and takes no value.
func ExampleSwitch() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleSwitch", facts,
		hello, pzm.Switch("loud", 'l', "says it louder"))
	tool.Run([]string{"hello"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello", "--loud"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello", "-l"}, os.Stdout, os.Stdout)
	// a switch given a value is refused, and nothing runs
	var errs strings.Builder
	code := tool.Run([]string{"hello", "--loud=yes"}, os.Stdout, &errs)
	fmt.Println(code, strings.Contains(errs.String(),
		"a switch takes no value"))
	// Output:
	// hello
	// HELLO
	// HELLO
	// 2 true
}

// A value is one word after the flag, or after its =; Given tells a
// value of nothing from no value at all.
func ExampleValue() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleValue", facts,
		func(call pzm.Call) error {
			if !call.Given("name") {
				return sayHello(call)
			}
			_, err := fmt.Fprintf(call.Out(), "hello, %s\n",
				call.Value("name"))
			return err
		}, pzm.Value("name", 'n', "who to say hello to"))
	tool.Run([]string{"hello", "--name", "ada"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello", "--name=bo"}, os.Stdout, os.Stdout)
	tool.Run([]string{"-n", "cy", "hello"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello"}, os.Stdout, os.Stdout)
	// Output:
	// hello, ada
	// hello, bo
	// hello, cy
	// hello
}

// A list is one value split on its commas; an empty value is an empty
// list.
func ExampleList() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleList", facts,
		func(call pzm.Call) error {
			for _, name := range call.List("to") {
				fmt.Fprintf(call.Out(), "hello, %s\n", name)
			}
			_, err := fmt.Fprintln(call.Out(), len(call.List("to")),
				"in all")
			return err
		}, pzm.List("to", verb.NoShort, "who to say hello to"))
	tool.Run([]string{"hello", "--to", "ada,bo"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello", "--to="}, os.Stdout, os.Stdout)
	// Output:
	// hello, ada
	// hello, bo
	// 2 in all
	// 0 in all
}

// A tool's own flags are every verb's, given before the verb or among
// its words.
func ExampleTool_Flags() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	tool.Flags(pzm.Value("lang", verb.NoShort, "the language to say it in"))
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleTool_Flags", facts,
		func(call pzm.Call) error {
			if call.Value("lang") == "fr" {
				_, err := fmt.Fprintln(call.Out(), "bonjour")
				return err
			}
			return sayHello(call)
		})
	tool.Run([]string{"--lang", "fr", "hello"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello", "--lang=fr"}, os.Stdout, os.Stdout)
	tool.Run([]string{"hello"}, os.Stdout, os.Stdout)
	// Output:
	// bonjour
	// bonjour
	// hello
}

// A verb reads what it is handed: standard input under Run, and what a
// test hands it under RunWith.
func ExampleCall_In() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleCall_In", facts,
		func(call pzm.Call) error {
			name, err := io.ReadAll(call.In())
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(call.Out(), "hello, %s\n",
				strings.TrimSpace(string(name)))
			return err
		})
	tool.RunWith([]string{"hello"}, strings.NewReader("ada\n"),
		os.Stdout, os.Stdout)
	// Output:
	// hello, ada
}

// RunWith is 0 when the verb ran, 1 when it failed, and 2 when nothing
// could run; a flag the parser will not take is refused in its own
// words.
func ExampleTool_RunWith() {
	tool := pzm.New("hi", "says hello", "", "", "", "")
	facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
		verb.ThisMachine, verb.NoOne, verb.Light)
	tool.Verb("hello", verb.NoShort, "says hello", "# hello", "ExampleTool_RunWith",
		facts, sayHello)
	tool.Verb("fail", verb.NoShort, "fails, always", "# fail", "ExampleTool_RunWith",
		facts, func(pzm.Call) error { return errors.New("no") })
	for _, args := range [][]string{{"hello"}, {"fail"},
		{"hello", "--bogus"}} {
		var out, errs strings.Builder
		code := tool.RunWith(args, strings.NewReader(""), &out, &errs)
		fmt.Printf("%d %q %q\n", code, out.String(), errs.String())
	}
	// Output:
	// 0 "hello\n" ""
	// 1 "" "hi fail: no\n"
	// 2 "" "hi: unknown flag: --bogus\n"
}
