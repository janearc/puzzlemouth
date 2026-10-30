# settings a tool's user sets

pzm has its own file: `~/.config/pzm/config`. it is the only file pzm
reads for settings, and it is the same for every tool built on pzm.

a file that is not there is fine: the defaults apply. a file with a line
pzm does not know, or a line wrong in its words, is refused whole. the
tool says which file and line, and runs nothing.

a tool's own config, `~/.config/NAME/config`, is the tool's alone. pzm
never opens it, whatever lines it holds.

    signer ada :: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA...
    help-width 70

to see whether the file is sound without running a verb, run
`NAME --who`: a refused file stops it, and it names the line.

## signer

`signer NAME :: KEY` says whose signature `--who` vouches for. KEY is
the line an authorized_keys file holds, such as the `.pub` file
ssh-keygen writes. give one line for each person. a name given twice is
refused, naming the second line.

with no signer line, `--who` ends with this line, and treats any
`NAME.sig` as unvouched:

    no signer is set: no key is vouched for; add a signer line to PATH

pzm has no built-in signer.

with a signer set, a signature by a key nobody vouches for is an
error. a binary with no signature is "not signed".

## help-width

`help-width N` is the most characters a line of help may have, for a
verb or a flag. N is a positive whole number: 0, -3, 7.5 and words are
refused, naming the line. with no line it is 70, which keeps a line of
help under 80 columns after a two-space margin.

pzm checks it when the tool is checked or run, so a wider width lets the
same tool declare longer help.
