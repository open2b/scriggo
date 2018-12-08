// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser_test

import (
	"log"
	"os"

	"open2b/template"
	"open2b/template/ast"
	"open2b/template/parser"
)

func ExampleParseSource() {

	src := []byte(`
  {% extends "layout.txt" %}
  {% macro Main %}
      {% for i, p in products %}
      {{ i }}. {{ p.Name }}: $ {{ p.Price }}
      {% end %}
  {% end macro %}`)

	tree, err := parser.ParseSource(src, ast.ContextText)
	if err != nil {
		log.Printf("parsing error: %s\n", err)
	}

	err = template.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleParser_Parse() {

	r := parser.DirReader("./template/")
	p := parser.New(r)

	tree, err := p.Parse("index.html", ast.ContextHTML)
	if err != nil {
		log.Printf("parsing error: %s\n", err)
	}

	err = template.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleMapReader() {

	r := parser.MapReader(map[string][]byte{
		"header.csv": []byte("Name"),
		"names.csv":  []byte("{% show `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
	})

	tree, err := r.Read("names.csv", ast.ContextText)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

	err = template.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleDirReader() {

	// Creates a reader that read sources from the files in the directory
	// "template/".

	r := parser.DirReader("template/")

	_, err := r.Read("index.html", ast.ContextHTML)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

}

func ExampleNewDirLimitedReader() {

	// Creates a reader that read sources from the files in the directory
	// "template/" and returns the error ErrReadTooLarge if a file to read
	// is larger than 50 KB or if all the read files exceed 256 KB.

	r := parser.NewDirLimitedReader("template/", 50*1024, 256*1204)

	_, err := r.Read("very-large-file.csv", ast.ContextText)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

}
