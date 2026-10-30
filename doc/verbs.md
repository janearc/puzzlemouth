# verbs

this is how a command built on pzm declares what it does, and what pzm
gives back. the design was decided on 2026-10-14. the end of this page
says what is built and what is not.

## the rule

the author's job is to make a tool not scary. the user should not have to
work out how to avoid setting something on fire. so everything a person
or an agent needs to use a verb safely is part of declaring it.

pzm refuses a part that is left out, or that cannot be what it says. it
refuses when the verb is made, and the tool does not run until it is put
right.

## what a verb declares

- its name: lowercase words joined by hyphens, as `foo-bar`. its long
  flag is the name, `--foo-bar`.
- its short flag: one letter or digit, only where it earns its letter.
- one line of help for a person: seventy characters or fewer. the design
  also asks that it pass okapi. that check is not built.
- its page in visualdocs: markdown, drawn by glamour. visualdocs is not
  built.
- any example function in its tests, with an `// Output:` comment. the
  tool's test calls `tool.Examples`, which finds the function and holds it
  to what it prints. visualdocs will show the function, so docs cannot
  drift. any file that go test leaves out on this build does not count.
- the example's name: `Example_fooBar` for the verb foo-bar. go vet
  refuses an example named for an identifier that is not there.
- its flags: each is a switch or a value or a list. `doc/flags.md` has the
  rules.
- its facts: typed and not prose. made by `verb.NewFacts`. `doc/facts.md`
  has the rules. they say:
  - where it writes: nothing, its state directory, its config directory,
    the working tree, or a named place.
  - whether it changes or deletes what is already there.
  - whether it reaches outside the machine.
  - whether it needs a person: prompt, passphrase or confirmation.
  - whether it can hold a cpu or the gpu at full load.

## what pzm gives back

status: `--help`, `--agent-help` and `--dry-run` are built. `--share` and
visualdocs are not.

- `--help` for a person, from the help lines and the flags.
- `--agent-help`, written by pzm from the facts. every tool has the same
  shape, so an agent learns it once. authors write no prose for
  agents.
- `--dry-run` on every verb that writes. its example must pass the flag
  to some call. pzm knows which verbs write from their facts. the check
  does not see which verb the call runs or what it writes. keeping to a
  dry run is the verb's job.
- `--share N` on every heavy verb. the verb's work is handed an idler and
  calls it once per unit of work. the idler sleeps in proportion to the
  work, so the verb uses N of a core's time. its workers scale with N.
- that is backoff. nice only reorders. the test helper runs the
  verb at a share of 0.25 and compares cpu time with wall time. any heavy
  verb that ignores its idler fails `game check`.
- visualdocs: the page through glamour in a small window, and the example
  through chroma in a larger one, paged.

## depth, args and text

status: none of this is built. every tool has verbs, and a verb's words are
`[ARGS]`.

- one level of scopes: `tool SCOPE VERB [ARGS] [FLAGS]`, as aws has
  `aws s3 cp`. each scope groups verbs and does nothing itself. it has no
  way to hold a scope, so a third level does not compile.
- the tool's flags apply everywhere and a scope's apply to its verbs. each
  verb's flags apply to itself. the verb answers to its flags inside its
  scope as at the top.
- the verb declares its args, so its help says `open NAME` and not
  `[ARGS]`. args are named places with at most one rest.
- the rest can be text: everything after the verb joined into one string.
  `pzmcli string-monch i do like to eat several spies a day` needs no
  quotes. flags stop where the text starts.
- the shell eats `'`, `*`, `#`, `;` and `$` before pzm sees them.
  the help of a text verb says to quote those, and `--agent-help` says always.

## config, and its seal

status: the config package is built. it has settings and a block one
level deep. it also reads a file in game's format. pzm reads a tool's
config, as doc/settings.md says.

`check-config`, `edit-config` and `seal` are not built. pzm refuses a
tool that names one.

- settings are declared like verbs. the file is read in game's format.
  each line is a key and its words. the tail follows ` :: ` with a space
  on each side. only the first one counts. `#` first on a line starts a
  comment. a block is indented with spaces, tabs or both.
- pzm bounces a file that does not conform, whole.
- every tool has `check-config [FILE]`. it says the file is sound, or it
  lists every bounce by line. it runs nothing and writes nothing.
- `seal` hashes the file and writes the seal as its last line, as a
  comment. there is no file beside it. people may sign instead, with
  `ssh-keygen -Y sign` in a config namespace. agents cannot sign. their
  seal is a hash with its session line.
- check-config says one of three things: sealed and unchanged, changed
  since it was sealed, or not sealed. it says "not sealed" and does not
  bounce. the tool may declare that its config must be sealed. the seal
  matters most on dotfiles, which no git is watching.
- `edit-config` follows visudo's pattern. it copies the file, opens
  `$EDITOR`, runs the check on exit, then seals the copy and renames it
  into place. it never installs a bounced copy.

## daemons are one binary with their tool

status: decided, and not built.

decided 2026-10-14: `moth serve` is a verb. `edit-config`,
`check-config`, `status`, `health` and `logs` are verbs of the same
binary. so the daemon and its editor read one set of declared settings
through one check and one seal. they cannot disagree about a sound file.

the daemon watches its config's directory, since editors save by renaming
over the file. it reloads a change with a good seal. it refuses a change
that is unsealed or has a seal that does not match. it runs on the last
good config and says so in status, health and its logs.

this will come up again when a daemon needs rights its editing side
should not have. it will also come up when a client elsewhere should not
carry the whole binary. then it is decided again, on purpose.

## why the parts are typed

each part is its own type and New takes them in order. no variable of one
type can be passed as another. untyped literals still convert, on
purpose. making each part a struct would mean wrapping every literal, and
that costs more than it saves.

## built, and not yet

built: New with name and short flag and help and page and example. Check holds
every example to its output. it refuses two verbs with one name or short
flag. Shown is the example as written. cobra is underneath and hidden
from every tool, and a test holds it hidden.

built: `--age` and `--who` from game's four stamps, and config read in
game's format. `--agent-help` writes a labelled line for the tool and for
each verb, from their declarations. it also says how the tool exits: 0
ran, 1 the verb failed, 2 nothing ran.

built: flags, the page for them in `doc/flags.md`, and examples for each
kind that run as tests. each verb declares its own with Switch, Value and
List. each tool declares flags every verb answers to. flags and words mix in
any order, and a lone `--` ends them.

after the verb the parser reads flags as it reads any flag. any value is
the next word whatever it looks like. short flags cluster. the parser
refuses a flag it will not take, in its own words.

before the verb pzm reads flags by the same rule, since a value must not
be taken for the verb. any flag of a verb given there must be the flag of
the verb that runs. otherwise pzm refuses the call and nothing runs.

pzm refuses a flag given twice or a switch given a value and an empty
item in a list. these go through the parser. any switch given its own long
form (`--loud=--loud`) counts as given, since that is what the parser
hands a bare one. pzm also refuses the tool's own flags with no verb.

called with nothing after it, a tool prints its help. it runs one verb
then, at a terminal, if the tool names one with Bare. `doc/bare.md` says
which verbs to name.

pzm keeps the names of what it gives, so a tool that declares one is
refused now. the names are `--dry-run`, `--share`, `check-config`,
`edit-config` and `seal`, beside `--help`, `--agent-help`, `--age` and
`--who`.

the verb asked for by its flag comes first, after the tool's own flags.
the verb reads standard input from its Call.

built: the facts, and the page for them in `doc/facts.md`. every verb
states all five, and pzm refuses one left out. `--agent-help` gives each
verb's facts a line. any verb that writes is given `--dry-run` and reads it
with call.DryRun. some call in its example must be given the flag.

not yet: declared arguments for a verb, `--share` and the idler, okapi on
the help line, and visualdocs.

later: provenance. pzm will report when something has gone around it,
such as a command that imports cobra or flag itself. nothing is refused
for it, since anyone may build what they like.
