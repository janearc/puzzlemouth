# puzzlemouth

## what is a.. what?

do you ever feel like you're talking to an agent and you don't understand what
it's saying? yeah, me too. it feels like sometimes i'm reading five or six
sentences of something trying to solve a riddle. at one point i called one of
them 'puzzlemouth' in exasperation and i decided i wanted to call this
package, which i'd been thinking about for a while, *puzzlemouth.* that is
ironically a bit long for me to be typing, so we just call it `pzm`
internally.

### but what does it do

so primarily pzm adds a bunch of things that i want all my applications to
include:

- support for `game` style config files
- support for uber-style mocks
- support for assistive reading devices

improved documentation:
- support for `visualdocs`, which is a nice little wrapper for docs that let
  you specify "pages" with code examples for everything in a package
- support for `--long-args` and `-sh -or -t -args` with rendered help for each
- support for `--agent-help` and `--help-claude` which allows agents to
  understand the tool they're working with

- provenance (who built this)
- versioning (what build is this)

and, importantly, these aren't optional; saying there is "support" implies you
can just build stuff without these. because i require all tooling on my
machine to include pzm, all tooling on my machine has these features from the
start. the rest is managed by `game` and vetted by `shuriken` and `okapi`.

this means that all tooling and products i build meat a minimum bar for the
functions i want in code quality and functionality, without having to ask an
agent to go back and *please, yes, i do mean add that before you push to main,
yes, i know, please do add that, thanks, i know, okay thanks, yes please do
add that,* and so on. it's fine.

## details, very briefly

puzzlemouth (pzm) is a go library for command-line tools. a tool declares its
verbs once, with facts and flags. pzm parses the call, runs it, explains it.

```go
facts := verb.NewFacts(verb.WritesNothing, verb.LeavesAlone,
	verb.ThisMachine, verb.NoOne, verb.Light)
tool := pzm.New("hi", "says hello", pzm.Build(build), pzm.Built(built),
	pzm.Who(who), pzm.Sessions(sids))
tool.Verb("hello", verb.NoShort, "says hello", page, "Example_hello", facts, hello,
	pzm.Switch("loud", 'l', "says it louder"))
os.Exit(tool.Run(os.Args[1:], os.Stdout, os.Stderr))
```

every tool gets `--help`, `--agent-help`, `--age` and `--who`. every verb
that writes gets `--dry-run`, and its example must pass that flag to some
call. pzm refuses a tool declared wrong. cobra parses underneath, hidden.

    doc/why.md      what each part is for, and the API in brief
    doc/verbs.md    the design: what a verb declares, what pzm gives
    doc/facts.md    what a verb does to the machine, and --dry-run
    doc/flags.md    flags: the three kinds, how they mix, what is refused
    doc/bare.md     what a tool does when called with nothing after it
    doc/settings.md signer, help-width: `signer ada :: ssh-ed25519 AAAA...`
