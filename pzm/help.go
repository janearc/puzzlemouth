package pzm

import (
	"fmt"
	"strings"
)

// builtinNames are the names pzm answers to itself, which no verb or
// flag may have.
var builtinNames = map[string]bool{"help": true, "agent-help": true,
	"age": true, "who": true}

// reserved are the names doc/verbs.md says pzm gives a tool, and to which
// verbs. --dry-run is built, and the rest are not yet. Declaring one
// shows the doc was not read, and it would clash with pzm's.
var reserved = map[string]string{
	"dry-run":      "--dry-run to every verb whose facts say it writes",
	"share":        "--share to every heavy verb",
	"check-config": "check-config to every tool",
	"edit-config":  "edit-config to every tool",
	"seal":         "seal to every tool",
}

// givenByPzm is why a verb or a flag may not be named for something pzm
// gives, or nothing when the name is not one of them.
func givenByPzm(name string) error {
	if gives, ok := reserved[name]; ok {
		return fmt.Errorf("pzm gives %s; see doc/verbs.md", gives)
	}
	return nil
}

// toolHelp is the tool's --help. It has the tool's line and a line for
// each verb, then the tool's own flags and how to call it.
func (tool *Tool) toolHelp() string {
	var page strings.Builder
	fmt.Fprintf(&page, "%s: %s\n\n", tool.name, tool.help)
	wide := none
	for _, each := range tool.verbs {
		name := each.verb.Name()
		wide = max(wide, len(name))
	}
	for _, each := range tool.verbs {
		fmt.Fprintf(&page, "  %-*s    %-5s%s\n", wide, each.verb.Name(),
			each.verb.Short(), each.verb.Help())
	}
	if some(tool.flags) {
		page.WriteString("\nfor every verb:\n" + flagLines(tool.flags))
	}
	fmt.Fprintf(&page, "\n%s VERB [ARGS], or %s --VERB [ARGS]\n", tool.name,
		tool.name)
	if bare := tool.bareVerb(); bare != "" {
		fmt.Fprintf(&page, "%s with nothing after it, at a "+
			"terminal, is %s %s\n", tool.name, tool.name, bare)
	}
	page.WriteString("--help for this, --agent-help for the same, " +
		"for an agent\n--age and --who for which build this is, " +
		"and who made it\n")
	return page.String()
}

// verbHelp is one verb's --help: what it does and every way to ask for
// it. Then come its flags and the tool's, which it answers to as well.
func (tool *Tool) verbHelp(each declaredVerb) string {
	also := fmt.Sprintf("%s %s", tool.name, each.verb.Long())
	if each.verb.Short() != "" {
		also += fmt.Sprintf(" or %s %s", tool.name, each.verb.Short())
	}
	page := fmt.Sprintf("%s %s [ARGS]: %s\nthe same as %s\n", tool.name,
		each.verb.Name(), each.verb.Help(), also) +
		flagLines(each.allFlags())
	if some(tool.flags) {
		page += "and for every verb:\n" + flagLines(tool.flags)
	}
	return page
}

// usage is a flag as it is given: its long form, its short one and what
// it takes. Switches show nothing after it and values show one word.
// Lists show two items with a comma.
func usage(flag Flag) string {
	said := flag.declared.Long()
	if flag.declared.Short() != "" {
		said += ", " + flag.declared.Short()
	}
	switch flag.kind {
	case valueKind:
		said += " VALUE"
	case listKind:
		said += " A,B"
	}
	return said
}

// flagLines is a line for each flag with its usage and its help in
// columns. It is nothing when there are no flags.
func flagLines(flags []Flag) string {
	wide := none
	for _, each := range flags {
		shown := usage(each)
		wide = max(wide, len(shown))
	}
	var lines strings.Builder
	for _, each := range flags {
		fmt.Fprintf(&lines, "  %-*s    %s\n", wide, usage(each),
			each.declared.Help())
	}
	return lines.String()
}

// agentHelp is the tool's --agent-help: one labelled line for each thing
// and how it exits. Each verb's facts have a line of their own.
func (tool *Tool) agentHelp() string {
	var page strings.Builder
	fmt.Fprintf(&page, "tool %s: %s\n", tool.name, tool.help)
	for _, toolFlag := range tool.flags {
		fmt.Fprintf(&page, "flag %s: %s\n", usage(toolFlag),
			toolFlag.declared.Help())
	}
	for _, each := range tool.allVerbs() {
		called := each.verb.Name()
		if each.verb.Short() != "" {
			called += " " + each.verb.Short()
		}
		fmt.Fprintf(&page, "verb %s: %s\n", called, each.verb.Help())
		fmt.Fprintf(&page, "facts %s: %s\n", each.verb.Name(),
			each.verb.Facts())
		for _, verbFlag := range each.allFlags() {
			fmt.Fprintf(&page, "flag %s %s: %s\n", each.verb.Name(),
				usage(verbFlag), verbFlag.declared.Help())
		}
	}
	if bare := tool.bareVerb(); bare != "" {
		fmt.Fprintf(&page, "bare %s: runs %s at a terminal, and "+
			"prints --help elsewhere\n", tool.name, bare)
	}
	fmt.Fprintf(&page, "call %s VERB [ARGS] or %s --VERB [ARGS]; exit 0 "+
		"ran, 1 the verb failed, 2 nothing ran\n", tool.name, tool.name)
	return page.String()
}
