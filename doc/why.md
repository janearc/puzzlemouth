# why pzm

go has flag, cobra and a dozen more. pzm exists because every tool
decided the same things again, a little differently each time. people
and agents had to learn each tool's way. each part of pzm answers one of
those decisions.

## every tool parsed its flags its own way

every verb declares its flags and pzm reads them. each flag is a switch, a value
or a list. it is long or short, and it mixes with words in any order. no tool
parses a flag by hand. `doc/flags.md` has the rules.

## which build is this, and who made it?

`--age` says which commit a binary was built from, and how long ago.
`--who` says which session built it, on which machine and which
sessions made the commits behind it. game stamps both in when it builds.

`--who` also checks the signature beside the binary. the binary makes
that check itself, so treat it as a convenience. `ssh-keygen -Y verify`
is the proof.

## agents read a tool for the first time

`--help` is for a person at a terminal. `--agent-help` gives the same
tool to an agent, one labelled line per thing. every tool has the same
shape, so an agent learns it once.

## what will this do to my machine?

every verb says where it writes. it says whether it changes or deletes
what is there, whether it leaves the machine, whether it needs a person
and how hard it works. `--agent-help` prints all of it.

every verb that writes gets `--dry-run`. the flag asks the verb to say
what it would do, and do none of it.

those facts are the author's word. pzm hands a verb the switch and
cannot see what it writes. the verb's example must pass `--dry-run` to
some call, and that is the one check. it does not see which verb the call
runs.

## docs drift, and examples run

every verb has a page and an example function with its output. go test
runs the example and holds it to the output it claims. the docs cannot
say one thing while the tool does another.

## tools declared wrong never run

leave out a part or give two verbs one name or declare a flag that pzm
gives every tool. the tool will not run. its tests call `tool.Check`,
which says why.

## the API in brief

    pzm.New        a tool: its name, its line of help, game's four stamps
    tool.Verb      a verb: name, short flag, help, page, example, facts,
                   what it does, and its flags
    verb.NewFacts  what a verb does to the machine; doc/facts.md
    tool.Flags     the tool's own flags, which every verb answers to
    pzm.Switch     a flag that is on when given: --loud
    pzm.Value      a flag that takes one value: --name ada
    pzm.List       a flag that takes a list, split on commas: --to ada,bo
    call.Args      the words after the verb, its flags taken out
    call.On        a switch; Value, List and Given read the others
    call.In        what the verb reads; call.Out is where it writes
    call.DryRun    whether to say what it would do, and do none of it
    tool.Run       runs the verb asked for; RunWith hands it its input
                   exit 0 is the verb ran, 1 it failed, 2 nothing ran
    tool.Check     everything declared wrong, for the tool's tests
    tool.Examples  Check, and every verb's example held to its output
    config.Read    settings in game's format, every one declared
