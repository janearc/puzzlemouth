# flags

every verb declares its flags when it is declared and pzm reads them. each tool
can declare flags of its own too, and every verb answers to those. no
tool parses a flag by hand.

```go
tool.Flags(pzm.Value("lang", verb.NoShort, "the language to say it in"))
tool.Verb("hello", verb.NoShort, "says hello", page, "Example_hello", facts, hello,
	pzm.Switch("loud", 'l', "says it louder"),
	pzm.Value("name", 'n', "who to say hello to"),
	pzm.List("to", verb.NoShort, "everyone to say hello to"))
```

## the three kinds

    kind     declared       given                     read
    switch   pzm.Switch     --loud, -l                call.On("loud")
    value    pzm.Value      --name ada, --name=ada,   call.Value("name")
                            -n ada, -nada
    list     pzm.List       --to ada,bo, --to=ada,bo  call.List("to")

`--help` shows a switch as `--loud, -l`, a value as `--name, -n VALUE`
and a list as `--to A,B`.

call.Given tells a value of nothing (`--name=`) from no value at all.
`--to=` gives a list that is empty. any list given as `--to a,,b`
is refused, since an empty item is never meant.

## flags and words

flags and words mix in any order. `hello ada --loud` and `hello --loud
ada` are the same call, and call.Args is `ada` either way. the lone `--`
ends the flags. everything after it is a word, however it looks.

any value is the next word whatever it looks like. `--name --loud` gives
the name `--loud` and leaves the switch off. any value that starts with a
hyphen reads best after =, as `--name=-x`.

short flags cluster. `-lq` is `-l -q`. any short value flag takes the rest
of its cluster as its value, so `-nada` is the name `ada`. `-ln ada` is
the switch and the name. that is the parser's way.

after the verb the parser reads the flags. before the verb pzm reads
them itself by the same rule. it must know which words are values to find
the verb.

the tool's own flags go before the verb or among its words. `hi --lang fr
hello` and `hi hello --lang=fr` are the same call. the verb can also be
asked for by its flag, `hi --hello`.

its short flag may run together with others. say a verb `look` has the
letter `l` and the tool has a switch `-q`. then `-ql` is `-q` and the
verb `look`.

the verb's own flag may go before the verb too. `hi -n ada hello` is `hi
hello -n ada`. pzm reads every word before the verb. the verb is the
first word that is not a flag or a flag's value. the flag's value is never
taken for the verb, whatever it says.

any flag given before the verb must be the flag of the verb that runs. when
two verbs declare a flag alike, either may follow it. when they declare
it differently, pzm cannot read it before the verb. it refuses the call.
put the flag after the verb.

verbs declare a flag differently when it is a switch to one and a value
to the other. they also do when one short letter is two long names, such
as `-n` for `--name` in one verb and `--number` in the other.

the lone `-` before the verb is a word for the verb, given to it first.
`hi - -n ada hello` gives hello the word `-`. the `--` after it ends the
flags before any verb is read. `hi - -- hello` is refused because `-` is
no verb.

## what is refused

any call that cannot be read runs nothing, and the tool exits 2. when the
parser will not take a flag, the tool says so in the parser's own words.
each call below is the sample's. under it is what the tool says, whole:

    hi hello --bogus
    hi: unknown flag: --bogus

    hi hello --name
    hi: flag needs an argument: --name

    hi hello --loud=yes
    hi: invalid argument "yes" for "-l, --loud" flag: a switch takes no value

    hi hello -l --loud
    hi: invalid argument "--loud" for "-l, --loud" flag: it is given twice

    hi hello --to ada,,bo
    hi: invalid argument "ada,,bo" for "--to" flag: "ada,,bo" has an empty item

the parser hands a bare switch its own long form. so the one value a
switch counts as given is that form: `--loud=--loud` is `--loud`. any
other value is refused.

any declaration that cannot be keeps the tool from running at all. Check
says why, for the tool's tests. pzm refuses:

- any name that is not lowercase words joined by hyphens.
- two flags of one verb with one name or one short flag.
- any tool's flag that is a verb's name or short flag or one of its flags.
- any flag not made by pzm.Switch, pzm.Value or pzm.List.
- any verb's flag that has the name or short flag of any verb. before the
  verb that word asks for the verb, so the flag could not be given.

any verb that reads a flag it did not declare or reads it as another kind
is a bug in the tool. it panics and names the verb and the flag.

## what pzm gives

pzm gives every tool these, so no tool declares them:

    --help         for a person: the verbs, their flags, how to call them
    --agent-help   the same for an agent, one labelled line per thing
    --age          which commit this build is, and how long ago it was made
    --who          who built it, and the sessions behind it
    -h             --help

every verb whose facts say it writes also gets this, as doc/facts.md
says:

    --dry-run      asks the verb to say what it would do, and do none of it

doc/verbs.md promises these too. they are not built yet. pzm keeps their
names, so a tool that declares one is refused now. that is better than a
tool that breaks when pzm's version lands:

    --share        on every heavy verb
    check-config   a verb every tool has
    edit-config    a verb every tool has
    seal           a verb every tool has

## examples

each of these is a test in pzm, so it cannot drift from what it shows:
Example_hello and ExampleSwitch and ExampleValue and ExampleList
and ExampleTool_Flags and ExampleCall_In and ExampleTool_RunWith. the sample at
the top of this page is Example_helloFlags, line for line.
