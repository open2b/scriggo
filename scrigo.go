// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strings"

	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

const (
	LimitMemorySize  Option = 1 << iota // limit allocable memory size.
	AllowShebangLine                    // allow shebang line; only for scripts.
)

type Option int

type PredefinedPackage = compiler.PredefinedPackage

type PredefinedConstant = compiler.PredefinedConstant

// Constant returns a constant, given its type and value, that can be used as
// a declaration in a predefined package.
//
// For untyped constants the type is nil.
func Constant(typ reflect.Type, value interface{}) PredefinedConstant {
	return compiler.Constant(typ, value)
}

type Program struct {
	fn      *vm.Function
	globals []compiler.Global
	options Option
}

// Load loads a program.
func Load(path string, reader Reader, packages map[string]*PredefinedPackage, options Option) (*Program, error) {
	p := newParser(reader, packages)
	tree, deps, err := p.parse(path)
	if err != nil {
		return nil, err
	}
	opts := &compiler.Options{
		IsPackage: true,
	}
	tci, err := compiler.Typecheck(opts, tree, nil, packages, deps, nil)
	if err != nil {
		return nil, err
	}
	typeInfos := map[ast.Node]*compiler.TypeInfo{}
	for _, pkgInfos := range tci {
		for node, ti := range pkgInfos.TypeInfo {
			typeInfos[node] = ti
		}
	}
	alloc := options&LimitMemorySize != 0

	pkgMain := compiler.EmitPackageMain(tree.Nodes[0].(*ast.Package), packages, typeInfos, tci[path].IndirectVars, alloc)
	globals := make([]compiler.Global, len(pkgMain.Globals))
	for i, global := range pkgMain.Globals {
		globals[i].Pkg = global.Pkg
		globals[i].Name = global.Name
		globals[i].Type = global.Type
		globals[i].Value = global.Value
	}
	return &Program{fn: pkgMain.Main, globals: globals, options: options}, nil
}

// Options returns the options with which the program has been loaded.
func (p *Program) Options() Option {
	return p.options
}

type RunOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	TraceFunc     vm.TraceFunc
}

// Run starts the program and waits for it to complete.
func (p *Program) Run(options RunOptions) error {
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if p.options&LimitMemorySize == 0 {
			return errors.New("program not loaded with LimitMemorySize option")
		}
		vmm.SetMaxMemory(options.MaxMemorySize)
	}
	if options.DontPanic {
		vmm.SetDontPanic(true)
	}
	if options.TraceFunc != nil {
		vmm.SetTraceFunc(options.TraceFunc)
	}
	if n := len(p.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range p.globals {
			if global.Value == nil {
				globals[i] = reflect.New(global.Type).Interface()
			} else {
				globals[i] = global.Value
			}
		}
		vmm.SetGlobals(globals)
	}
	_, err := vmm.Run(p.fn)
	return err
}

// Disassemble disassembles the package with the given path. Predefined
// packages can not be disassembled.
func (p *Program) Disassemble(w io.Writer, pkgPath string) (int64, error) {
	packages, err := compiler.Disassemble(p.fn)
	if err != nil {
		return 0, err
	}
	asm, ok := packages[pkgPath]
	if !ok {
		return 0, errors.New("package path does not exist")
	}
	n, err := io.WriteString(w, asm)
	return int64(n), err
}

// Global represents a global variable with a package, name, type (only for
// not predefined globals) and value (only for predefined globals). Value, if
// present, must be a pointer to the variable value.
type Global struct {
	Name string
	Type reflect.Type
}

type Script struct {
	fn      *vm.Function
	globals []Global
	options Option
}

// LoadScript loads a script from a reader.
func LoadScript(src io.Reader, main *PredefinedPackage, options Option) (*Script, error) {

	alloc := options&LimitMemorySize != 0
	shebang := options&AllowShebangLine != 0

	// Parsing.
	buf, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	tree, _, err := compiler.ParseSource(buf, false, shebang)
	if err != nil {
		return nil, err
	}

	opts := &compiler.Options{
		IsPackage: false,
	}
	tci, err := compiler.Typecheck(opts, tree, main, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// Emitting.
	// TODO(Gianluca): pass "main" to emitter.
	// main contains user defined variables.
	emitter := compiler.NewEmitter(nil, tci["main"].TypeInfo, tci["main"].IndirectVars)
	emitter.SetAlloc(alloc)
	fn := compiler.NewFunction("main", "main", reflect.FuncOf(nil, nil, false))
	emitter.CurrentFunction = fn
	emitter.FB = compiler.NewBuilder(emitter.CurrentFunction)
	emitter.FB.SetAlloc(alloc)
	emitter.FB.EnterScope()
	compiler.AddExplicitReturn(tree)
	emitter.EmitNodes(tree.Nodes)
	emitter.FB.ExitScope()
	emitter.FB.End()

	return &Script{fn: emitter.CurrentFunction, options: options}, nil
}

// Options returns the options with which the script has been loaded.
func (s *Script) Options() Option {
	return s.options
}

// Run starts a script with the specified global variables and waits for it to
// complete.
func (s *Script) Run(vars map[string]interface{}, options RunOptions) error {
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if s.options&LimitMemorySize == 0 {
			return errors.New("script not loaded with LimitMemorySize option")
		}
		vmm.SetMaxMemory(options.MaxMemorySize)
	}
	if options.DontPanic {
		vmm.SetDontPanic(true)
	}
	if options.TraceFunc != nil {
		vmm.SetTraceFunc(options.TraceFunc)
	}
	if n := len(s.globals); n > 0 {
		globals := make([]interface{}, n)
		for i, global := range s.globals {
			if value, ok := vars[global.Name]; ok {
				if v, ok := value.(reflect.Value); ok {
					globals[i] = v.Addr().Interface()
				} else {
					rv := reflect.New(global.Type).Elem()
					rv.Set(reflect.ValueOf(v))
					globals[i] = rv.Interface()
				}
			} else {
				globals[i] = reflect.New(global.Type).Interface()
			}
		}
		vmm.SetGlobals(globals)
	}
	_, err := vmm.Run(s.fn)
	return err
}

// Disassemble disassembles a script.
func (s *Script) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, s.fn)
}

// parser implements a parser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are compiler.Cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the parser is no longer used,
// because it would be the compiler.Cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type parser struct {
	reader   compiler.Reader
	packages map[string]*PredefinedPackage
	trees    *compiler.Cache
	// TODO (Gianluca): does packageInfos need synchronized access?
	packageInfos map[string]*compiler.PackageInfo // key is path.
	typeCheck    bool
}

// newParser returns a new parser that reads the trees from the reader r. typeCheck
// indicates if a type-checking must be done after parsing.
func newParser(r compiler.Reader, packages map[string]*PredefinedPackage) *parser {
	p := &parser{
		reader:   r,
		packages: packages,
		trees:    &compiler.Cache{},
	}
	return p
}

// parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// parse is safe for concurrent use.
func (p *parser) parse(path string) (*ast.Tree, compiler.GlobalsDependencies, error) {

	// Path must be absolute.
	if path == "" {
		return nil, nil, compiler.ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := compiler.ToAbsolutePath("/", path)
	if err != nil {
		return nil, nil, err
	}

	pp := &expansion{p.reader, p.trees, p.packages, []string{}}

	tree, deps, err := pp.parsePath(path)
	if err != nil {
		if err2, ok := err.(*compiler.SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(compiler.CycleError); ok {
			err = compiler.CycleError(path + "\n\t" + string(err2))
		}
		return nil, nil, err
	}
	if len(tree.Nodes) == 0 {
		return nil, nil, &compiler.SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("expected 'package' or script, found 'EOF'")}
	}

	return tree, deps, nil
}

// TypeCheckInfos returns the type-checking infos collected during
// type-checking.
func (p *parser) typeCheckInfos() map[string]*compiler.PackageInfo {
	return p.packageInfos
}

// expansion is an expansion state.
type expansion struct {
	reader   compiler.Reader
	trees    *compiler.Cache
	packages map[string]*PredefinedPackage
	paths    []string
}

// abs returns path as absolute.
func (pp *expansion) abs(path string) (string, error) {
	var err error
	if path[0] == '/' {
		path, err = compiler.ToAbsolutePath("/", path[1:])
	} else {
		parent := pp.paths[len(pp.paths)-1]
		dir := parent[:strings.LastIndex(parent, "/")+1]
		path, err = compiler.ToAbsolutePath(dir, path)
	}
	return path, err
}

// parsePath parses the source at path in context ctx. path must be absolute
// and cleared.
func (pp *expansion) parsePath(path string) (*ast.Tree, compiler.GlobalsDependencies, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, nil, compiler.CycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ast.ContextGo); ok {
		return tree, nil, nil
	}
	defer pp.trees.Done(path, ast.ContextGo)

	src, err := pp.reader.Read(path)
	if err != nil {
		return nil, nil, err
	}

	tree, deps, err := compiler.ParseSource(src, true, false)
	if err != nil {
		return nil, nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	expandedDeps, err := pp.expand(tree.Nodes)
	if err != nil {
		if e, ok := err.(*compiler.SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, nil, err
	}
	for k, v := range expandedDeps {
		deps[k] = v
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the compiler.Cache.
	pp.trees.Add(path, ast.ContextGo, tree)

	return tree, deps, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *expansion) expand(nodes []ast.Node) (compiler.GlobalsDependencies, error) {

	allDeps := compiler.GlobalsDependencies{}

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.Package:

			deps, err := pp.expand(n.Declarations)
			if err != nil {
				return nil, err
			}
			for k, v := range deps {
				allDeps[k] = v
			}

		case *ast.Import:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return nil, err
			}
			found := false
			for path := range pp.packages {
				if path == n.Path {
					found = true
					break
				}
			}
			if found {
				continue
			}
			var deps compiler.GlobalsDependencies
			n.Tree, deps, err = pp.parsePath(absPath + ".go")
			if err != nil {
				if err == compiler.ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == compiler.ErrNotExist {
					err = &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("cannot find package \"%s\"", n.Path)}
				} else if err2, ok := err.(compiler.CycleError); ok {
					err = compiler.CycleError("imports " + string(err2))
				}
				return nil, err
			}
			for k, v := range deps {
				allDeps[k] = v
			}

		}

	}

	return allDeps, nil
}
