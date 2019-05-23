// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"scrigo/internal/compiler/ast"
)

// Makes a dependency analysis on packages during parsing.
// See https://golang.org/ref/spec#Package_initialization for further
// informations.

type GlobalsDependencies map[*ast.Identifier][]*ast.Identifier

// dependencies analyzes dependencis between global declarations.
type dependencies struct {

	// pending is the list of global declarations waiting for evaluation.
	pending []*ast.Identifier

	// deps keeps track of dependencies between global declarations.
	deps map[*ast.Identifier]map[*ast.Identifier]int

	// scopes tracks variable and constants declaration inside scopes which
	// can shadow global declarations.
	scopes []map[string]struct{}

	// lastLhs keeps track of last multiple declaration. It's used when number
	// of lhs identifiers is greater than rhs'one.
	lastLhs []*ast.Identifier
}

// result returns all dependencies of global declarations.
func (d *dependencies) result() GlobalsDependencies {
	if d == nil {
		return nil
	}
	out := GlobalsDependencies{}
	for glob, deps := range d.deps {
		out[glob] = []*ast.Identifier{}
		for d, count := range deps {
			if count > 0 {
				out[glob] = append(out[glob], d)
			}
		}
	}
	return out
}

// enterScope enters a new scope.
func (d *dependencies) enterScope() {
	if d == nil {
		return
	}
	d.scopes = append(d.scopes, map[string]struct{}{})
}

// exitScope exits current scope.
func (d *dependencies) exitScope() {
	if d == nil {
		return
	}
	d.scopes = d.scopes[:len(d.scopes)-1]
}

// declare sets lhs as declared.
func (d *dependencies) declare(lhs []*ast.Identifier) {
	if d == nil {
		return
	}
	if len(d.scopes) == 0 {
		// Global declaration.
		if d == nil {
			return
		}
		if d.deps == nil {
			d.deps = map[*ast.Identifier]map[*ast.Identifier]int{}
		}
		if len(d.pending) > 0 {
			panic(fmt.Errorf("called registerGlobals when there are still pending globals (%v)", d.pending))
		}
		// If len(lhs) is greater than one, parser is parsing a multiple
		// assignment (var or const). If len(rhs) is one all left symbols share
		// the same dependency set.
		if len(lhs) > 1 {
			d.lastLhs = lhs
		}
		d.pending = lhs
	} else {
		// Local declaration.
		for _, left := range lhs {
			d.scopes[len(d.scopes)-1][left.Name] = struct{}{}
			// When parser finds "var a = 10" inside a scope, a is parsed as identifier
			// then added as dependency of current global declaration, reporting a false-positive.
			curr := d.pending[0]
			for dep := range d.deps[curr] {
				if dep.Name == left.Name {
					d.deps[curr][dep]--
				}
			}
		}
	}
}

// end ends dependency evaluation of current global declaration.
func (d *dependencies) end() {
	if d == nil {
		return
	}
	if len(d.scopes) > 0 {
		panic("cannot call end() with opened scopes")
	}
	if len(d.pending) > 0 {
		// If ending global declaration has no dependencies, its identifier
		// is added to the result with an empty depencency set.
		curr := d.pending[0]
		if len(d.deps[curr]) == 0 {
			d.deps[curr] = nil
		}
		d.pending = d.pending[1:]
	}
}

// endList ends evaluation of global variables and constants. endList must be
// called even when the number of lhs is one.
func (d *dependencies) endList() {
	if d == nil {
		return
	}
	// If there are still pending global declarations, parser is parsing a
	// multiple assignment (var or const). All left symbols must share the
	// same dependency set.
	if len(d.pending) > 0 {
		firstLeft := d.lastLhs[0]
		deps := d.deps[firstLeft]
		for _, ident := range d.lastLhs[1:] {
			d.deps[ident] = deps
		}
	}
	d.pending = nil
}

// uses sets name as used by current global declaration, which will make
// dinstiction between global declarations (that introduce a dependency) and
// local declarations (that do not).
func (d *dependencies) use(name *ast.Identifier) {
	if d == nil {
		return
	}
	if len(d.pending) == 0 {
		return
	}
	if isBlankIdentifier(name) {
		return
	}
	// If name is local defined, it's not a global dependency.
	for i := len(d.scopes) - 1; i >= 0; i-- {
		if _, ok := d.scopes[i][name.Name]; ok {
			return
		}
	}
	// Instead of adding and identifier with the same name, increments count
	// of current one.
	curr := d.pending[0]
	for dep := range d.deps[curr] {
		if dep.Name == name.Name {
			d.deps[curr][dep]++
			return
		}
	}
	if d.deps[curr] == nil {
		d.deps[curr] = map[*ast.Identifier]int{}
	}
	d.deps[curr][name]++
}
