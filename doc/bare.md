# bare

called with nothing after it, a tool prints its help. any tool can name one
of its verbs to run then instead. it does so at a terminal with Bare:

```go
tool.Verb("look", verb.NoShort, "says what is there", page, "Example_look", facts,
	look)
tool.Bare("look")
tool.Run(nil, os.Stdout, os.Stderr)
```

so `peek` typed at a terminal is `peek look`. its help says so:

    peek VERB [ARGS], or peek --VERB [ARGS]
    peek with nothing after it, at a terminal, is peek look

`--agent-help` has a line for it:

    bare peek: runs look at a terminal, and prints --help elsewhere

## at a terminal

the verb runs only when the call's input and output are both terminals.
anywhere else the call prints the help. that covers a pipe or a file or
/dev/null or an agent's shell. tools with no Bare do the same.

people type the name at a terminal and get the verb. agents run a
tool bare to learn what it is. they get the usage. they do not get a
screen that waits for keys it will never send.

the output of a test is never a terminal. any test of a bare call says what to
take it for:

    tool.AtTerminal(func(io.Reader, io.Writer) bool { return true })

## which verb

name a verb that is safe to run by accident. people type a tool's name to
see what it is as often as to use it. the bare call is the one they least
meant to aim.

- verbs that only read are safe. they say what they see.
- verbs like a tuner are safe. they open a screen and write only as the
  keys there ask.
- verbs that write as soon as they run are not safe. switchers, syncs
  and pushes are examples. give those a name to be typed.

## what does not change

only a call with nothing at all after the tool runs the verb. the rest is
read as it always was:

    peek --help         the help, as always
    peek --agent-help   the agent's help
    peek look           look, by its name
    peek stray          refused: there is no verb stray
    peek --in attic     refused: the tool's own flags with no verb

the verb's exit codes are its own. 0 means it ran and 1 means it failed.
any call that cannot be read exits 2.

## what is refused

Check refuses a Bare that names no verb of the tool's. pzm's own verbs
count as no verb of the tool's. Check also refuses a second Bare. the tool
declared so runs nothing and exits 2, as any tool declared wrong does.
the verb may be declared before Bare or after.
