package verb

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"strconv"
)

// sample is one example function in a command's tests. held is whether
// go test compares what it prints, and dry whether it runs something dry.
// code is the function as it is written. left is the file it is in when
// go test does not compile that file here, and nothing when it does.
type sample struct {
	held bool
	dry  bool
	code string
	left string
}

// samples are a command's example functions, by function name.
type samples map[string]sample

// Check is what a command's tests call, with the command's own test
// files. It holds every verb to an example that is there and prints what
// it claims, so go test runs it and compares the output. It also refuses
// two verbs that share a name or a short flag.
//
// Any example in a file go test leaves out on this build does not count.
//
// The dry-run check for a verb that writes is syntactic. It wants a call
// in the example given "--dry-run" and does not see what that call runs.
func Check(source fs.FS, verbs ...*Verb) error {
	found, err := read(source)
	if err != nil {
		return err
	}
	var refused []error
	names, shorts := map[Name]bool{}, map[Short]bool{}
	for _, each := range verbs {
		if names[each.name] {
			refused = append(refused,
				fmt.Errorf("two verbs are named %s", each.name))
		}
		if each.short != NoShort && shorts[each.short] {
			letter := rune(each.short)
			refused = append(refused,
				fmt.Errorf("two verbs have -%c", letter))
		}
		names[each.name], shorts[each.short] = true, true
		if err := found.holds(each); err != nil {
			refused = append(refused, err)
		}
	}
	return errors.Join(refused...)
}

// holds is nothing when a verb's example is in the tests and holds its
// output, and otherwise why not.
func (found samples) holds(made *Verb) error {
	each, err := found.of(made)
	if err != nil {
		return err
	}
	if !each.held {
		return fmt.Errorf("verb %s: %s has no Output comment, so go "+
			"test never compares what it prints", made.name,
			made.example)
	}
	if made.facts.WritesAnything() && !each.dry {
		return fmt.Errorf("verb %s: it writes %s, so %s must pass "+
			"\"--dry-run\" to some call, and none does", made.name,
			made.facts.writes, made.example)
	}
	return nil
}

// runsDry is whether a function makes a call one of whose arguments
// gives "--dry-run": the string itself, or a list that holds it. Naming
// it in a comment is not that. Nor is a string set aside, even inside a
// function handed to a call.
func runsDry(function *ast.FuncDecl) bool {
	search := &dryCalls{}
	ast.Inspect(function.Body, search.visit)
	return search.found
}

// dryCalls looks through a function for a call that gives "--dry-run".
type dryCalls struct{ found bool }

// visit is ast.Inspect's step. It stops once the call is found.
func (search *dryCalls) visit(node ast.Node) bool {
	if call, isCall := node.(*ast.CallExpr); isCall {
		for _, arg := range call.Args {
			search.found = search.found || givesDryRun(arg)
		}
	}
	return !search.found
}

// givesDryRun is whether an argument is "--dry-run" or a list that holds
// it. It does not look into a function the argument holds.
func givesDryRun(arg ast.Expr) bool {
	search := &dryStrings{}
	ast.Inspect(arg, search.visit)
	return search.found
}

// dryStrings looks through an argument for the string "--dry-run".
type dryStrings struct{ found bool }

// visit is ast.Inspect's step. It does not look into a function.
func (search *dryStrings) visit(node ast.Node) bool {
	switch each := node.(type) {
	case *ast.FuncLit:
		return false
	case *ast.BasicLit:
		text, err := strconv.Unquote(each.Value)
		search.found = search.found || each.Kind == token.STRING &&
			err == nil && text == "--dry-run"
	}
	return !search.found
}

// of is a verb's example, or why it cannot be checked or shown. It is
// not in the tests. Or its file is one go test leaves out.
func (found samples) of(made *Verb) (*sample, error) {
	each, there := found[string(made.example)]
	switch {
	case !there:
		return nil, fmt.Errorf("verb %s: %s is not in the tests",
			made.name, made.example)
	case each.left != "":
		return nil, fmt.Errorf("verb %s: %s is in %s, which "+
			"go test leaves out on this build, so it never runs",
			made.name, made.example, each.left)
	}
	return &each, nil
}

// Shown is the whole function of a verb's example, as written in the
// source, for the code window in visualdocs.
func Shown(source fs.FS, made *Verb) (string, error) {
	found, err := read(source)
	if err != nil {
		return "", err
	}
	each, err := found.of(made)
	if err != nil {
		return "", err
	}
	return each.code, nil
}

// read is the example functions in every _test.go file at the top of
// source, found the way go test finds them. fs.Glob fails only on a
// malformed pattern. This one is a constant, so its error is nought.
//
// Examples in a file go test leaves out are marked, since they never run.
func read(source fs.FS) (samples, error) {
	names, _ := fs.Glob(source, "*_test.go")
	found := samples{}
	for _, name := range names {
		raw, err := fs.ReadFile(source, name)
		if err != nil {
			return nil, err
		}
		compiled, err := compiles(source, name)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if err := found.add(name, raw, compiled); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// compiles is whether go test compiles the file on this build. Go skips
// some names and others are for another system. Constraints are those of
// the build this runs in. Tags given to go test are not known, so a file
// that needs one is left out.
//
// That includes race, which go test -race sets. Its examples are refused
// as ones go test leaves out.
func compiles(source fs.FS, name string) (bool, error) {
	context := build.Default
	context.OpenFile = opener{source}.open
	return context.MatchFile(".", name)
}

// opener opens the files go/build asks for from a file system.
type opener struct{ files fs.FS }

// open is a file of the file system, for go/build.
func (from opener) open(path string) (io.ReadCloser, error) {
	return from.files.Open(path)
}

// add is the examples in one test file: go/doc says which functions are
// examples and whether each holds its output, and the file says how each
// is written.
//
// Examples from a compiled file replace those from a file left out, and
// never the reverse. Each system may have its own.
func (found samples) add(name string, raw []byte, compiled bool) error {
	places := token.NewFileSet()
	file, err := parser.ParseFile(places, name, raw, parser.ParseComments)
	if err != nil {
		return err
	}
	here := samples{}
	for _, each := range doc.Examples(file) {
		held := each.Output != "" || each.EmptyOutput
		here["Example"+each.Name] = sample{held: held}
	}
	for _, declared := range file.Decls {
		function, isFunction := declared.(*ast.FuncDecl)
		if !isFunction || function.Recv != nil {
			continue
		}
		each, there := here[function.Name.Name]
		if !there {
			continue
		}
		from := places.Position(function.Pos()).Offset
		to := places.Position(function.End()).Offset
		each.code = string(raw[from:to])
		each.dry = runsDry(function)
		here[function.Name.Name] = each
	}
	for key, each := range here {
		if _, there := found[key]; !compiled && there {
			continue
		}
		if !compiled {
			each.left = name
		}
		found[key] = each
	}
	return nil
}
