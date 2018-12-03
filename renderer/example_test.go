// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package renderer_test

import (
	"log"
	"os"

	"open2b/template/ast"
	"open2b/template/parser"
	"open2b/template/renderer"
)

func ExampleRender() {

	p := parser.New(parser.DirReader("/home/salinger/book/"))

	tree, err := p.Parse("cover.html", ast.ContextHTML)
	if err != nil {
		log.Fatalf("parsing error: %s", err)
	}

	vars := map[string]string{"title": "The Catcher in the Rye"}

	err = renderer.RenderTree(os.Stdout, tree, vars, nil)
	if err != nil {
		log.Fatalf("rendering error: %s", err)
	}

}

func ExampleRender_2() {

	p := parser.New(parser.DirReader("/home/salinger/book/"))

	tree, err := p.Parse("index.html", ast.ContextHTML)
	if err != nil {
		log.Fatalf("parsing error: %s", err)
	}

	vars := map[string]string{"title": "The Catcher in the Rye"}

	err = renderer.RenderTree(os.Stdout, tree, vars, func(err error) bool {
		log.Printf("rendering error: %s\n", err)
		return true
	})
	if err != nil {
		log.Fatalf("rendering error: %s", err)
	}

}
