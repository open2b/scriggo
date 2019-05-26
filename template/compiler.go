// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
	"scrigo/vm"
)

// Context indicates the type of source that has to be rendered and controls
// how to escape the values to render.
type Context int

const (
	ContextNone   Context = Context(ast.ContextNone)
	ContextText   Context = Context(ast.ContextText)
	ContextHTML   Context = Context(ast.ContextHTML)
	ContextCSS    Context = Context(ast.ContextCSS)
	ContextScript Context = Context(ast.ContextScript)
)

type Option int

const (
	LimitMemorySize Option = 1 << iota
)

type RenderOptions struct {
	Context       context.Context
	MaxMemorySize int
	DontPanic     bool
	TraceFunc     vm.TraceFunc
}

type Template struct {
	main    *scrigo.PredefinedPackage
	fn      *vm.Function
	options Option
}

func Load(path string, reader Reader, main *scrigo.PredefinedPackage, ctx Context, options Option) (*Template, error) {

	// Parsing.
	p := newParser(reader)
	tree, err := p.parse(path, main, ast.Context(ctx))
	if err != nil {
		return nil, convertError(err)
	}

	opts := &compiler.Options{
		IsPackage: false,
	}
	tci, err := compiler.Typecheck(opts, tree, main, nil, nil, tcBuiltins)
	if err != nil {
		return nil, err
	}

	alloc := options&LimitMemorySize != 0

	// Emitting.
	// TODO(Gianluca): pass "main" and "builtins" to emitter.
	// main contains user defined variabiles, while builtins contains template builtins.
	// define something like "emitterBuiltins" in order to avoid converting at every compilation.
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

	return &Template{main: main, fn: emitter.CurrentFunction}, nil
}

func (t *Template) Render(out io.Writer, vars map[string]reflect.Value, options RenderOptions) error {
	// TODO: implement globals
	vmm := vm.New()
	if options.Context != nil {
		vmm.SetContext(options.Context)
	}
	if options.MaxMemorySize > 0 {
		if t.options&LimitMemorySize == 0 {
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
	_, err := vmm.Run(t.fn)
	return err
}

// Options returns the options with which the template has been loaded.
func (t *Template) Options() Option {
	return t.options
}

// Disassemble disassembles a template.
func (t *Template) Disassemble(w io.Writer) (int64, error) {
	return compiler.DisassembleFunction(w, t.fn)
}

func convertError(err error) error {
	if err == compiler.ErrInvalidPath {
		return compiler.ErrInvalidPath
	}
	if err == compiler.ErrNotExist {
		return compiler.ErrNotExist
	}
	return err
}

// parser implements a parser that reads the tree from a Reader and expands
// the nodes Extends, Import and Include. The trees are cached so only one
// call per combination of path and context is made to the reader even if
// several goroutines parse the same paths at the same time.
//
// Returned trees can only be transformed if the parser is no longer used,
// because it would be the cached trees to be transformed and a data race can
// occur. In case, use the function Clone in the astutil package to create a
// clone of the tree and then transform the clone.
type parser struct {
	reader compiler.Reader
	trees  *compiler.Cache
	// TODO (Gianluca): does packageInfos need synchronized access?

	// TODO(Gianluca): deprecated, remove.
	packageInfos map[string]*compiler.PackageInfo // key is path.
}

// newParser returns a new Parser that reads the trees from the reader r.
// typeCheck indicates if a type-checking must be done after parsing.
func newParser(r compiler.Reader) *parser {
	return &parser{
		reader:       r,
		trees:        &compiler.Cache{},
		packageInfos: make(map[string]*compiler.PackageInfo),
	}
}

// parse reads the source at path, with the reader, in the ctx context,
// expands the nodes Extends, Import and Include and returns the expanded tree.
//
// Parse is safe for concurrent use.
func (p *parser) parse(path string, main *scrigo.PredefinedPackage, ctx ast.Context) (*ast.Tree, error) {

	// Path must be absolute.
	if path == "" {
		return nil, compiler.ErrInvalidPath
	}
	if path[0] == '/' {
		path = path[1:]
	}
	// Cleans the path by removing "..".
	path, err := compiler.ToAbsolutePath("/", path)
	if err != nil {
		return nil, err
	}

	pp := &expansion{p.reader, p.trees, map[string]*scrigo.PredefinedPackage{"main": main}, []string{}}

	tree, err := pp.parsePath(path, ctx)
	if err != nil {
		if err2, ok := err.(*compiler.SyntaxError); ok && err2.Path == "" {
			err2.Path = path
		} else if err2, ok := err.(compiler.CycleError); ok {
			err = compiler.CycleError(path + "\n\t" + string(err2))
		}
		return nil, err
	}

	if len(tree.Nodes) == 0 {
		return nil, &compiler.SyntaxError{"", ast.Position{1, 1, 0, 0}, fmt.Errorf("??????????, found 'EOF'")}
	}

	return tree, nil
}

// expansion is an expansion state.
type expansion struct {
	reader   compiler.Reader
	trees    *compiler.Cache
	packages map[string]*scrigo.PredefinedPackage
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
func (pp *expansion) parsePath(path string, ctx ast.Context) (*ast.Tree, error) {

	// Checks if there is a cycle.
	for _, p := range pp.paths {
		if p == path {
			return nil, compiler.CycleError(path)
		}
	}

	// Checks if it has already been parsed.
	if tree, ok := pp.trees.Get(path, ctx); ok {
		return tree, nil
	}
	defer pp.trees.Done(path, ctx)

	src, err := pp.reader.Read(path, ctx)
	if err != nil {
		return nil, err
	}

	tree, _, err := compiler.ParseSource(src, false, ctx)
	if err != nil {
		return nil, err
	}
	tree.Path = path

	// Expands the nodes.
	pp.paths = append(pp.paths, path)
	err = pp.expand(tree.Nodes, ctx)
	if err != nil {
		if e, ok := err.(*compiler.SyntaxError); ok && e.Path == "" {
			e.Path = path
		}
		return nil, err
	}
	pp.paths = pp.paths[:len(pp.paths)-1]

	// Adds the tree to the cache.
	pp.trees.Add(path, ctx, tree)

	return tree, nil
}

// expand expands the nodes parsing the sub-trees in context ctx.
func (pp *expansion) expand(nodes []ast.Node, ctx ast.Context) error {

	for _, node := range nodes {

		switch n := node.(type) {

		case *ast.If:

			for {
				err := pp.expand(n.Then.Nodes, ctx)
				if err != nil {
					return err
				}
				switch e := n.Else.(type) {
				case *ast.If:
					n = e
					continue
				case *ast.Block:
					err := pp.expand(e.Nodes, ctx)
					if err != nil {
						return err
					}
				}
				break
			}

		case *ast.For:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.ForRange:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Switch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.TypeSwitch:

			var err error
			for _, c := range n.Cases {
				err = pp.expand(c.Body, ctx)
				if err != nil {
					return err
				}
			}

		case *ast.Macro:

			err := pp.expand(n.Body, ctx)
			if err != nil {
				return err
			}

		case *ast.Extends:

			if len(pp.paths) > 1 {
				return &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("extended, imported and included paths can not have extends")}
			}
			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == compiler.ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == compiler.ErrNotExist {
					err = &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("extends path %q does not exist", absPath)}
				} else if err2, ok := err.(compiler.CycleError); ok {
					err = compiler.CycleError("imports " + string(err2))
				}
				return err
			}

		case *ast.Import:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == compiler.ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == compiler.ErrNotExist {
					err = &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("import path %q does not exist", absPath)}
				} else if err2, ok := err.(compiler.CycleError); ok {
					err = compiler.CycleError("imports " + string(err2))
				}
				return err
			}

		case *ast.Include:

			absPath, err := pp.abs(n.Path)
			if err != nil {
				return err
			}
			n.Tree, err = pp.parsePath(absPath, n.Context)
			if err != nil {
				if err == compiler.ErrInvalidPath {
					err = fmt.Errorf("invalid path %q at %s", n.Path, n.Pos())
				} else if err == compiler.ErrNotExist {
					err = &compiler.SyntaxError{"", *(n.Pos()), fmt.Errorf("included path %q does not exist", absPath)}
				} else if err2, ok := err.(compiler.CycleError); ok {
					err = compiler.CycleError("include " + string(err2))
				}
				return err
			}

		// TODO: to remove.
		default:
			panic(fmt.Errorf("unexpected node %s", node))

		}

	}

	return nil
}
