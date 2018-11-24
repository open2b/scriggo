// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package util implements methods to walk and dump a tree.
package util

import (
	"errors"
	"fmt"
	"io"
	"open2b"
	"strconv"

	"open2b/template/ast"
)

type dumper struct {
	output             io.Writer
	indentLevel        int
	externalReferences []*ast.Tree
}

type errVisitor struct {
	err error
}

func (e errVisitor) Error() string {
	return e.err.Error()
}

// Visit elaborates a node of a tree, writing on a Writer the representation
// of the same node correctly indented. The Visit method is called by the Walk
// function.
func (d *dumper) Visit(node ast.Node) Visitor {

	// Management of the v.Visit (nil) call made by Walk.
	if node == nil {
		d.indentLevel-- // Risale l'albero.
		return nil
	}

	d.indentLevel++

	// In the case where node contains a reference to another tree (ie node is
	// of type Import, Extend, ShowPath), it adds the reference to the
	// externalReferences list to visit the nodes with a recursive call of the
	// Dump.
	var tree *ast.Tree
	switch n := node.(type) {
	case *ast.Import:
		tree = n.Tree
	case *ast.Extend:
		tree = n.Tree
	case *ast.ShowPath:
		tree = n.Tree
	default:
		// No reference to add, it continues without doing anything.
	}
	if tree != nil {
		d.externalReferences = append(d.externalReferences, tree)
	}

	// If the node is of type Tree, it writes it and returns without doing anything else.
	if n, ok := node.(*ast.Tree); ok {
		_, err := fmt.Fprintf(d.output, "\nTree: %v:%v\n", strconv.Quote(n.Path), n.Position)
		if err != nil {
			panic(errVisitor{err})
		}
		return d
	}

	// Look for the representation as a node string. If the case is not defined
	// here, the default string conversion of the node is used.
	var text string
	switch n := node.(type) {
	case *ast.Text:
		text = n.Text
		if len(text) > 30 {
			text = open2b.Truncate(text, 30) + "..."
		}
		text = strconv.Quote(text)
	case *ast.If:
		text = n.Expr.String()
	default:
		text = fmt.Sprintf("%v", node)
	}

	// Inserts the right level of indentation.
	for i := 0; i < d.indentLevel; i++ {
		_, err := fmt.Fprint(d.output, "│    ")
		if err != nil {
			panic(errVisitor{err})
		}
	}

	// Determines the type by removing the prefix "*ast."
	typeStr := fmt.Sprintf("%T", node)[5:]

	posStr := node.Pos().String()

	_, err := fmt.Fprintf(d.output, "%v (%v) %v\n", typeStr, posStr, text)
	if err != nil {
		panic(errVisitor{err})
	}

	return d
}

// Dump writes the tree dump on w. In the case where the tree is nil,
// the function stops execution by returning an error other than nil.
// In case the tree is not extended, the function ends its execution
// after writing the base tree, returning a non-nil error.
func Dump(w io.Writer, node ast.Node) (err error) {

	defer func() {
		if r := recover(); r != nil {
			if t, ok := r.(errVisitor); ok {
				err = t.err
			} else {
				panic(r)
			}
		}
	}()

	if node == nil {
		return errors.New("can't dump a nil tree")
	}

	d := dumper{w, -1, []*ast.Tree{}}
	Walk(&d, node)

	// Writes trees that have been declared as external references.
	for _, tree := range d.externalReferences {
		Dump(w, tree)
	}

	return nil
}
