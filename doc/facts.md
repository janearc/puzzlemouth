# facts

every verb says what it does to the machine it runs on. the facts are
typed, so every tool says them the same way. pzm reads them. any verb that
writes gets `--dry-run`. `--agent-help` gives each verb's facts a line of
their own.

```go
facts := verb.NewFacts(verb.WritesState, verb.Deletes,
	verb.ThisMachine, verb.NoOne, verb.Light)
tool.Verb("tidy", verb.NoShort, "takes away what is old", page, "Example_tidy",
	facts, tidy)
tool.Run([]string{"tidy", "--dry-run"}, os.Stdout, os.Stderr)
```

## the five facts

`verb.NewFacts` takes them in this order. each has its own type. two
named values given out of order do not compile.

    type       what it says                    given as
    Writes     where it writes                 WritesNothing, WritesState,
                                               WritesConfig, WritesTree,
                                               WritesTo("a place")
    Existing   what it does to what is there   LeavesAlone, Changes,
                                               Deletes
    Reach      whether it leaves the machine   ThisMachine, Outside
    Person     whether it needs a person       NoOne, Prompt, Passphrase,
                                               Confirmation
    Load       how hard it works the machine   Light, FullCPU, FullGPU

untyped strings convert to Writes. untyped numbers convert to any
of the other four. the type refuses a conversion that is none of its values.

every part must be stated. pzm refuses a part left out, and a value that
is none of the above. the refusal comes when the verb is made, and the
tool does not run. pzm also refuses a verb that writes nothing and says
it changes or deletes what is there.

each part takes one value: the most the verb can do. any verb that writes
in two places names both with WritesTo. for example "its state
directory and the working tree".

any verb that may ask a person more than one thing gives the one that holds
them longest. the passphrase outranks the confirmation, and the confirmation
outranks the prompt. any verb that can hold the gpu says FullGPU.

WritesTo names any other place in a few words on one line. "the agents'
homes" and "a slack channel" are examples. WritesTo("nothing") is
WritesNothing, so name a real place.

the place may not hold a semicolon, because `--agent-help` puts semicolons
between the facts. join two places with "and".

any place on another machine is also Outside. slack channels are one.

## --dry-run is the verb's promise

any verb that writes anything but WritesNothing is given `--dry-run`. the
verb asks `call.DryRun()`. on a dry run it says what it would do and does
none of it.

pzm hands the verb the switch and cannot see what the verb writes. the
flag's line asks the verb. it does not promise for it. any verb that never
asks `call.DryRun()` writes for real under `--dry-run`. the facts are the
author's word, and so is keeping to a dry run.

the one check is the verb's example. pzm refuses a verb that writes
unless some call in its example is given `"--dry-run"`, alone or in a
list. any comment that names the flag does not count. any string that is set
aside does not count either.

the check does not see which verb the call runs. the example's reader
does.

no verb declares its own `--dry-run`. pzm refuses one that does. any verb
that writes nothing has no `--dry-run`. the parser refuses the flag there
in its own words: `unknown flag: --dry-run`.

## where the facts are said

`--agent-help` gives each verb's facts after its line, and its flags
after them:

    verb tidy: takes away what is old
    facts tidy: writes its state directory; may delete what is there;
      stays on this machine; needs no one; light on the machine
    flag tidy --dry-run: asks the verb to say what it would do, and do
      none of it

the real output is one line each. they are folded here to fit the page.
`--help` for a verb shows its help line and its flags, `--dry-run` among
them. it does not show the facts.

## not yet

`--share` from a verb's Load is not built, and neither is the idler
behind it. doc/verbs.md says how they will work.

## examples

Example_tidy is the sample at the top of this page, line for line
to its dry run. ExampleFacts shows facts in words. each runs as a test.
