// Package pzm is a command and its verbs, declared once and then parsed
// and run by pzm. Tools never see the parser underneath. So the parser
// can change without any tool changing.
package pzm

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"

	"github.com/janearc/puzzlemouth/verb"
	"golang.org/x/term"
)

// Do is what a verb does when it runs.
type Do func(call Call) error

// Call is one run of a verb. It holds the arguments after the verb and
// the flags it was given, what it reads and where it writes.
type Call struct {
	args  []string
	in    io.Reader
	out   io.Writer
	verb  string
	flags map[string]*flagValue
}

// Args are the arguments the verb was given, its flags taken out.
// Everything after a lone -- is an argument, however it looks.
func (call Call) Args() []string { return call.args }

// In is what the verb reads: the tool's standard input, or what a test
// gave RunWith.
func (call Call) In() io.Reader { return call.in }

// Out is where the verb writes what it has to say.
func (call Call) Out() io.Writer { return call.out }

// Tool is a command and its verbs. It keeps anything wrong with how they
// were declared until it is run.
//
// Declare everything before the first run. Declaring during a run is not
// supported, and pzm takes no lock for it.
type Tool struct {
	name    verb.Name
	help    verb.Help
	stamps  stamps
	settled *settled // what its config said, once it has been read
	flags   []Flag
	verbs   []declaredVerb
	refused []error
	bare    []verb.Name // what Bare named, each time it was called
	// executable and terminal are decided once, by Run, at the edge:
	// where the running binary is, and whether a call reads and writes
	// terminals. Nothing below Run asks the system again.
	executable string
	terminal   bool
	// atTerminal replaces terminal for a test; nil keeps terminal
	atTerminal func(in io.Reader, out io.Writer) bool
}

// declaredVerb is a verb as a tool declared it: the verb and its flags,
// and what it does.
type declaredVerb struct {
	verb  verb.Verb
	flags []Flag
	do    Do
}

// New is a tool with no verbs yet. It takes the tool's name and line of
// help, and the four stamps game builds into a command's main package.
// Plain go build leaves them empty or "dev", and --age and --who say so.
func New(name verb.Name, help verb.Help, build Build, built Built,
	who Who, sids Sessions) *Tool {
	return &Tool{name: name, help: help,
		stamps: stamps{build, built, who, sids}}
}

// Verb declares one of the tool's verbs, its facts and its flags. When
// the verb cannot be, the refusal is kept and the tool does not run until
// it is put right. Every verb whose facts say it writes is given
// --dry-run.
func (tool *Tool) Verb(name verb.Name, short verb.Short, help verb.Help,
	page verb.Page, example verb.Example, facts verb.Facts, do Do,
	flags ...Flag) {
	// Help is held to the configured width when the tool is checked, as
	// the config is not read until then.
	declared, err := verb.NewWithin(verb.Unlimited, name, short, help, page,
		example, facts)
	if err != nil {
		tool.refused = append(tool.refused, err)
		return
	}
	if err := tool.verbRefusal(*declared, do); err != nil {
		tool.refused = append(tool.refused, err)
		return
	}
	if err := flagsRefusal(nil, flags); err != nil {
		tool.refused = append(tool.refused, fmt.Errorf("verb %s: %w",
			name, err))
		return
	}
	// The tool keeps its own copy: flags is the caller's to change.
	tool.verbs = append(tool.verbs,
		declaredVerb{*declared, slices.Clone(flags), do})
}

// Bare names the verb a tool runs when nothing follows its name at a
// terminal. That is a call whose input and output are both terminals.
// Anywhere else a bare call prints the help, as a tool with no Bare
// does. --help always prints it.
//
// Typed by a person at a terminal, the name gets the verb. Run bare by
// an agent to learn what the tool is, it gets the usage.
//
// Name a verb that is safe to run by accident. One that only reads is
// safe. So is a tuner that opens a screen and writes only when its keys
// ask.
//
// The verb must be one of the tool's own. Check refuses any other name,
// and a second Bare.
func (tool *Tool) Bare(name verb.Name) {
	tool.bare = append(tool.bare, name)
}

// AtTerminal replaces how the tool tells whether a call is at a
// terminal. It is for tests: their input and output are never one. at
// is asked of RunWith's in and out. It is asked only for a call with no
// words, and only when Bare named a verb. Bare matters to nothing else.
func (tool *Tool) AtTerminal(at func(in io.Reader, out io.Writer) bool) {
	tool.atTerminal = at
}

// terminals is whether in and out are both a terminal: each an open file
// that is one. /dev/null, a pipe and a file are not, and neither is
// anything that is no file. The second is not asked if the first is not.
func terminals(in io.Reader, out io.Writer) bool {
	return isTerminal(in) && isTerminal(out)
}

// isTerminal is whether an end of a call is an open file that is a
// terminal.
func isTerminal(end any) bool {
	file, isFile := end.(*os.File)
	return isFile && term.IsTerminal(int(file.Fd()))
}

// bareHere is the verb a call with nothing after the tool runs here. At
// a terminal that is the one Bare named; anywhere else it is "".
func (tool *Tool) bareHere(in io.Reader, out io.Writer) verb.Name {
	bare := tool.bareVerb()
	if bare == "" {
		return ""
	}
	here := tool.terminal
	if tool.atTerminal != nil {
		here = tool.atTerminal(in, out)
	}
	if !here {
		return ""
	}
	return bare
}

// bareRefusals are what is wrong with what Bare named: a second Bare, or
// a name that is none of the tool's verbs.
func (tool *Tool) bareRefusals() []error {
	var refused []error
	if times := len(tool.bare); times > one {
		called := fmt.Errorf("pzm.Bare was called "+
			"%d times; a tool runs one verb bare", times)
		refused = append(refused, called)
	}
	for _, name := range tool.bare {
		if tool.declares(name) {
			continue
		}
		refused = append(refused, fmt.Errorf("pzm.Bare names %s, "+
			"which is no verb of %s's", name, tool.name))
	}
	return refused
}

// declares is whether one of the tool's own verbs has the name.
func (tool *Tool) declares(name verb.Name) bool {
	return slices.ContainsFunc(tool.verbs, named{name}.has)
}

// bareVerb is the verb the tool runs bare, or "" when it prints its help.
func (tool *Tool) bareVerb() verb.Name {
	if !some(tool.bare) {
		return ""
	}
	return tool.bare[first]
}

// Flags declares the tool's own flags. Every verb answers to them, given
// before the verb or among its words. It may be called more than once;
// each call is checked against the flags already declared.
func (tool *Tool) Flags(flags ...Flag) {
	if err := flagsRefusal(tool.flags, flags); err != nil {
		tool.refused = append(tool.refused, fmt.Errorf("tool %s: %w",
			tool.name, err))
		return
	}
	tool.flags = append(tool.flags, flags...)
}

// verbRefusal is why a verb that is sound on its own still cannot be one
// of this tool's. It has nothing to do, or a name pzm gives or keeps. Or
// its name or short flag is pzm's or another verb's.
func (tool *Tool) verbRefusal(declared verb.Verb, do Do) error {
	name := declared.Name()
	if err := givenByPzm(name); err != nil {
		return fmt.Errorf("verb %s: %w", name, err)
	}
	switch {
	case do == nil:
		return fmt.Errorf("verb %s: it does nothing; give it a Do",
			name)
	case builtinNames[name]:
		return fmt.Errorf("verb %s: the name is pzm's own", name)
	case declared.Short() == "-h":
		return fmt.Errorf("verb %s: -h is pzm's own, for help", name)
	}
	for _, other := range tool.verbs {
		short := declared.Short()
		sameShort := short != "" && other.verb.Short() == short
		if other.verb.Name() == name || sameShort {
			return fmt.Errorf("verb %s: its name or short flag "+
				"is %s's", name, other.verb.Name())
		}
	}
	return nil
}

// flagsRefusal is why flags cannot be declared after those already
// declared. Each flag is held to flagRefusals against the flags before
// it, already declared or not.
func flagsRefusal(already, flags []Flag) error {
	var refused []error
	for i, each := range flags {
		before := slices.Concat(already, flags[:i])
		refused = append(refused, flagRefusals(each, before)...)
	}
	return errors.Join(refused...)
}

// flagRefusals is why one flag cannot be declared beside those before
// it.
func flagRefusals(each Flag, before []Flag) []error {
	switch {
	case each.err != nil:
		return []error{each.err}
	case each.declared.Name() == "":
		return []error{errors.New("a flag not made by pzm.Switch, " +
			"pzm.Value or pzm.List")}
	}
	long := each.declared.Long()
	var refused []error
	if err := givenByPzm(each.declared.Name()); err != nil {
		refused = append(refused, fmt.Errorf("flag %s: %w", long, err))
	}
	if builtinNames[each.declared.Name()] ||
		each.declared.Short() == "-h" {
		refused = append(refused, fmt.Errorf("flag %s: the name or "+
			"short flag is pzm's own", long))
	}
	for _, other := range before {
		if clash(each, other) {
			beside := other.declared.Long()
			refused = append(refused, fmt.Errorf("flag %s: its "+
				"name or short flag is %s's", long, beside))
		}
	}
	return refused
}

// clash is whether two flags share a name or a short flag. Every caller
// asks of a sound flag. One that could not be made has no name and no
// short flag, so it clashes with no sound one.
func clash(one, other Flag) bool {
	short := one.declared.Short()
	return one.declared.Name() == other.declared.Name() ||
		short != "" && short == other.declared.Short()
}

// Check is everything wrong with how the tool's verbs and flags were
// declared, or nothing; a tool's tests call it.
func (tool *Tool) Check() error {
	refused := append(slices.Clone(tool.refused), tool.clashes()...)
	if set, err := tool.settings(); err != nil {
		refused = append(refused, err)
	} else {
		refused = append(refused, tool.helpRefusals(set.width)...)
	}
	bare := tool.bareRefusals()
	return errors.Join(append(refused, bare...)...)
}

// clashes is each of the tool's own flags that shares a name or a short
// flag with a verb or with one of a verb's flags. The tool's flags are
// every verb's, and asking for one must say which.
func (tool *Tool) clashes() []error {
	var refused []error
	for _, toolFlag := range tool.flags {
		for _, each := range tool.verbs {
			clashed := clashesWith(toolFlag, each)
			refused = append(refused, clashed...)
		}
	}
	for _, each := range tool.verbs {
		for _, verbFlag := range each.flags {
			asks := tool.asksFor(verbFlag, each)
			refused = append(refused, asks...)
		}
	}
	return refused
}

// asksFor is how a verb's flag clashes with the flags that ask for a
// verb: it has a verb's name or a verb's short flag. Before the verb
// pzm reads such a word as asking for that verb, so the flag could not
// be given there.
func (tool *Tool) asksFor(verbFlag Flag, owner declaredVerb) []error {
	var refused []error
	for _, each := range tool.verbs {
		if asks(verbFlag, each) {
			why := askedFor(owner, verbFlag, each)
			refused = append(refused, why)
		}
	}
	return refused
}

// asks is whether the flag has the name or the short flag of the verb.
func asks(flag Flag, each declaredVerb) bool {
	short := flag.declared.Short()
	return flag.declared.Name() == each.verb.Name() ||
		short != "" && short == each.verb.Short()
}

// askedFor is the refusal of a flag of owner's that asks for a verb.
func askedFor(owner declaredVerb, flag Flag, each declaredVerb) error {
	return fmt.Errorf("verb %s: flag %s has the name or short flag that "+
		"asks for verb %s", owner.verb.Name(), flag.declared.Long(),
		each.verb.Name())
}

// clashesWith is how one of the tool's flags clashes with one verb: its
// name or short flag, or one of its flags.
func clashesWith(toolFlag Flag, each declaredVerb) []error {
	var refused []error
	long, short := toolFlag.declared.Long(), toolFlag.declared.Short()
	if toolFlag.declared.Name() == each.verb.Name() ||
		short != "" && short == each.verb.Short() {
		name := each.verb.Name()
		refused = append(refused, fmt.Errorf("flag %s: its name or "+
			"short flag is verb %s's", long, name))
	}
	for _, verbFlag := range each.flags {
		if clash(toolFlag, verbFlag) {
			name, own := each.verb.Name(), verbFlag.declared.Long()
			refused = append(refused, fmt.Errorf("flag %s: verb "+
				"%s has its own %s", long, name, own))
		}
	}
	return refused
}

// Examples is Check, and verb.Check over the tool's verbs with the tool's
// own test files. Every verb's example must be there, and go test holds
// it to what it prints. The tool's tests call it with os.DirFS(".").
func (tool *Tool) Examples(tests fs.FS) error {
	verbs := make([]*verb.Verb, none, len(tool.verbs))
	for place := range tool.verbs {
		verbs = append(verbs, &tool.verbs[place].verb)
	}
	return errors.Join(tool.Check(), verb.Check(tests, verbs...))
}

// Run is RunWith, the verb reading the tool's standard input. It is the
// edge: it asks the system once where the running binary is and whether
// standard input and out are terminals, and the tool is handed those
// plain values.
//
// A tool that cannot tell where it is running is refused with exit 2.
func (tool *Tool) Run(args []string, out, errs io.Writer) int {
	path, err := os.Executable()
	return tool.start(args, os.Stdin, out, errs, path, err)
}

// start is Run with what the edge found handed in: the path of the
// running binary, or why it is not known.
func (tool *Tool) start(args []string, in io.Reader, out,
	errs io.Writer, path string, unknown error) int {
	if unknown != nil {
		fmt.Fprintf(errs, "%s: cannot tell where it is running: %v\n",
			tool.name, unknown)
		return statusUnread
	}
	tool.executable = path
	tool.terminal = terminals(in, out)
	return tool.RunWith(args, in, out, errs)
}

// RunWith runs the verb args ask for, reading in and writing to out and
// errs. Its result is 0 when the verb ran and 1 when the verb failed. It
// is 2 when nothing could be run, such as for a flag given wrong. Help
// that could not be written to out is the same.
//
// The parser's own words refuse a flag it will not take.
//
// With nothing after its name at a terminal the tool runs the verb Bare
// named.
func (tool *Tool) RunWith(args []string, in io.Reader, out,
	errs io.Writer) int {
	if err := tool.Check(); err != nil {
		fmt.Fprintf(errs, "%s: %v\n", tool.name, err)
		return statusUnread
	}
	if len(args) == none {
		if bare := tool.bareHere(in, out); bare != "" {
			args = []string{string(bare)}
		}
	}
	var unwritten error
	root := tool.root(in, out, errs, &unwritten)
	words, err := tool.verbFirst(args)
	if err == nil {
		root.SetArgs(words)
		err = root.Execute()
	}
	if err == nil {
		err = unwritten
	}
	var failing failed
	switch {
	case err == nil:
		return statusRan
	case errors.As(err, &failing):
		fmt.Fprintf(errs, "%s %s: %v\n", tool.name, failing.name,
			failing)
		return statusFailed
	default:
		fmt.Fprintf(errs, "%s: %v\n", tool.name, err)
		return statusUnread
	}
}

// failed is a verb that ran and failed, told apart from a tool that could
// not run one.
type failed struct {
	name string
	err  error
}

// Error is why the verb failed.
func (failing failed) Error() string { return failing.err.Error() }

// named is a verb name, to find the tool's own verb that has it.
type named struct{ name verb.Name }

// has is whether the declared verb has the name.
func (looked named) has(each declaredVerb) bool {
	return each.verb.Name() == string(looked.name)
}
