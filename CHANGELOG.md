# changelog

## unreleased

fixed:

- a block's lines may be indented with tabs, spaces or both, as game
  has always read them; v0.1.2 refused a mix. only an indented line
  under no block is refused.

## v0.1.2

added:

- `config.NewSchema[File]() (*Schema[File], error)` makes, once, what a
  file of type File may say, from the config and help tags on its
  fields. it refuses every mistake in the declaration together, each
  named by its field, and returns nil.
- `config.Schema.Read(name, raw, into)` fills `into`, or refuses the
  whole file with every bad line named as `name:line: key: why` and
  leaves `into` as it was. what `into` held before are the defaults.
- `config.Schema.Settings()` lists the keys, with their words and help,
  in field order, for help pages and syntax files.
- `config.Line` and `config.LineSetter`: a field type that means more
  than its words reads its own line.
- `config.Setting` is now a plain struct with exported fields.
- a block may be indented with tabs, as with spaces. a tab mixed with
  spaces in one indent is refused, and so is a line that changes between
  tabs and spaces inside one block.
- `yamlfile.Read[T](name, raw, into)` reads a YAML file into a type, on
  gopkg.in/yaml.v3 with unknown keys refused. every refusal is at its
  true line number, and the file is named once.
- `verb.Width.SetLine` reads a width from a config line, for a field of
  type `verb.Width`.

changed, breaking:

- the whole config API is gone. `config.New`, `config.Block`,
  `Setting.UpTo`, `config.Unlimited`, `config.Words`, `config.Key`,
  `config.Help`, `config.Takes`, `config.Tail`, `config.NoTail`,
  `config.Entry` and its getters, and the old `config.Read` are removed.

  a tool declares a struct and calls `Schema.Read`. the older
  `config.Setting` was a different type; the name now belongs to the
  plain struct above.
- numbers is rewritten on yamlfile.
  - `numbers.Numbers` is a `[]Number`. `Number` and `Source` have
    exported fields, and the getters are removed (`Name()`, `Value()`,
    `Unit()`, `Says()`, `Kind()`, `From()`, `By()`, and `Source`'s `Key()`,
    `Where()`, `Page()`, `Text()`).
  - `Number.From` is a `*Source`, nil for a picked number.
  - `numbers.Read` returns `Numbers`, nil with the error on a refusal.
    every refusal is reported together and the file is named once.
  - `Numbers.Get` returns `(*Number, error)`, a pointer into the list,
    nil on a refusal. its error no longer names the file.
  - `Numbers.All` is removed: range over the list.
  - the names `yes`, `on`, `no`, `off`, `true`, `false`, `y` and `n` are
    no longer refused. yaml.v3 keeps them as written. a name that parses
    as a number in any base is still refused.
  - a file with an unknown key is now refused at its true line; the
    refusal comes from yamlfile and is reported alone.
  - go.uber.org/config is no longer a dependency.
- the host package is gone. `internal/host` and `host.Machine` are
  removed, so nothing outside pzm used them. `pzm.AtTerminal` stays.
- constructors that can fail return a pointer, nil with the error:
  `verb.New`, `verb.NewWithin` and `verb.NewFlag` return `*Verb` and
  `*Flag`. `verb.Check` and `verb.Shown` take `*Verb`.
- a config line that sets `help-width` twice, or `signer` for one name
  twice, is refused at its line with `file:line: key: why`. the text of
  every pzm config refusal changed to that form.

removed : `go.uber.org/config` and `go.uber.org/mock` as requirements, and the
`mocks` run in `.game`.
