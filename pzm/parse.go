package pzm

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// root is the tool as cobra holds it for one run. Each verb is a command
// with its own flags. Each verb is a flag on the tool too, long and short.
// The tool's own flags are on every verb.
//
// The parser asks for help through a function that returns nothing. So
// help that cannot be written is left in unwritten.
func (tool *Tool) root(in io.Reader, out, errs io.Writer,
	unwritten *error) *cobra.Command {
	root := &cobra.Command{Use: string(tool.name), Short: string(tool.help),
		Args: cobra.ArbitraryArgs, SilenceUsage: true,
		SilenceErrors: true}
	root.SetOut(out)
	root.SetErr(errs)
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpFunc(helper{tool, nil, unwritten}.help)
	toolValues := declare(root.PersistentFlags(), tool.flags)
	asked := map[string]*bool{}
	for _, each := range tool.allVerbs() {
		root.AddCommand(tool.command(each, in, toolValues, unwritten))
		name, help := each.verb.Name(), each.verb.Help()
		short := strings.TrimPrefix(each.verb.Short(), "-")
		asked[name] = root.Flags().BoolP(name, short, false, help)
	}
	agent := root.Flags().Bool("agent-help", false, "")
	root.RunE = rootRun{tool, asked, agent, toolValues}.run
	return root
}

// toolFlagsAlone refuses the tool's own flags given with no verb to
// answer to them: they are for a verb, and none ran.
func toolFlagsAlone(toolValues map[string]*flagValue) error {
	var named []string
	for _, value := range toolValues {
		if value.given {
			named = append(named, value.flag.declared.Long())
		}
	}
	if !some(named) {
		return nil
	}
	slices.Sort(named)
	return fmt.Errorf("%s given, and no verb: a verb comes after "+
		"them", strings.Join(named, ", "))
}

// declare puts flags on the parser's flag set for one run. It is what
// each flag was given, by name.
func declare(set *pflag.FlagSet, flags []Flag) map[string]*flagValue {
	values := map[string]*flagValue{}
	for _, each := range flags {
		value := &flagValue{flag: each}
		values[each.declared.Name()] = value
		short := strings.TrimPrefix(each.declared.Short(), "-")
		parsed := set.VarPF(value, each.declared.Name(), short,
			each.declared.Help())
		if each.kind == switchKind {
			parsed.NoOptDefVal = each.declared.Long()
		}
	}
	return values
}

// command is one verb as cobra holds it. It has its flags and the tool's
// and its own help.
func (tool *Tool) command(each declaredVerb, in io.Reader,
	toolValues map[string]*flagValue, unwritten *error) *cobra.Command {
	command := &cobra.Command{Use: each.verb.Name(),
		Short: each.verb.Help(), Args: cobra.ArbitraryArgs}
	command.SetHelpFunc(helper{tool, &each, unwritten}.help)
	values := declare(command.Flags(), each.allFlags())
	for name, value := range toolValues {
		values[name] = value
	}
	command.RunE = verbRun{each, in, values}.run
	return command
}

// helper is the help of one command. It is one verb's when verb is set
// and the tool's when it is not.
type helper struct {
	tool      *Tool
	verb      *declaredVerb
	unwritten *error
}

// help writes the command's help. The parser's help function returns
// nothing, so a page that cannot be written is left in unwritten.
func (writes helper) help(asking *cobra.Command, _ []string) {
	out := asking.OutOrStdout()
	if _, err := fmt.Fprint(out, writes.page()); err != nil {
		*writes.unwritten = err
	}
}

// page is the help to write.
func (writes helper) page() string {
	if writes.verb == nil {
		return writes.tool.toolHelp()
	}
	return writes.tool.verbHelp(*writes.verb)
}

// run is the verb run as called, its failure kept as a verb's.
func (each declaredVerb) run(call Call) error {
	if err := each.do(call); err != nil {
		return failed{each.verb.Name(), err}
	}
	return nil
}

// noVerbFirst is the tool run with no verb named first. pzm read no verb
// before the words it left to the parser. Those words are help, a lone
// -- or a word that is no verb. Whatever follows them is left too.
//
// It prints the agent's help when that is asked. It prints the tool's
// help when nothing is. Any word that is no verb is refused.
//
// So is a flag that asks for a verb after one of those words. pzm reads
// such a flag as the verb only before them. The tool's own flags with
// nothing else are refused as well.
func (tool *Tool) noVerbFirst(asking *cobra.Command, args []string,
	asked map[string]*bool, agent bool,
	toolValues map[string]*flagValue) error {
	var chosen []declaredVerb
	for _, each := range tool.allVerbs() {
		if *asked[each.verb.Name()] {
			chosen = append(chosen, each)
		}
	}
	switch {
	case len(chosen) > one || agent && some(chosen):
		return errors.New("one verb at a time")
	case agent:
		_, err := fmt.Fprint(asking.OutOrStdout(), tool.agentHelp())
		return err
	case len(chosen) == one:
		return fmt.Errorf("%s comes first: %s %s [ARGS]",
			chosen[first].verb.Long(), tool.name,
			chosen[first].verb.Long())
	case some(args):
		return fmt.Errorf("there is no verb %s", args[first])
	}
	if err := toolFlagsAlone(toolValues); err != nil {
		return err
	}
	return asking.Help()
}

// allVerbs is the tool's verbs and pzm's own, which answer the same
// ways.
func (tool *Tool) allVerbs() []declaredVerb {
	return append(slices.Clone(tool.verbs), tool.builtinVerbs()...)
}

// rootRun is what the root command does when no verb is named: it
// reports what is wrong with the call, or shows the help.
type rootRun struct {
	tool       *Tool
	asked      map[string]*bool
	agent      *bool
	toolValues map[string]*flagValue
}

// run is rootRun as the command runs it.
func (root rootRun) run(asking *cobra.Command, args []string) error {
	return root.tool.noVerbFirst(asking, args, root.asked, *root.agent,
		root.toolValues)
}

// verbRun is what a verb's command does when it is asked for.
type verbRun struct {
	each   declaredVerb
	in     io.Reader
	values map[string]*flagValue
}

// run is verbRun as the command runs it.
func (named verbRun) run(asking *cobra.Command, args []string) error {
	return named.each.run(Call{args: args, in: named.in,
		out: asking.OutOrStdout(), verb: named.each.verb.Name(),
		flags: named.values})
}
