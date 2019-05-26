//+build ignore

// TODO(Gianluca): this file has been excluded from building. Consider what to
// modify and what to remove.

// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template

import (
	"log"
	"os"

	"scrigo"
	"scrigo/internal/compiler"
	"scrigo/internal/compiler/ast"
)

func ExampleParseSource() {

	src := []byte(`
  {% extends "layout.txt" %}
  {% macro Main %}
      {% for i, p in products %}
      {{ i }}. {{ p.Name }}: $ {{ p.Price }}
      {% end %}
  {% end macro %}`)

	tree, err := compiler.ParseSource(src, ast.ContextText, false)
	if err != nil {
		log.Printf("parsing error: %s\n", err)
	}

	err = scrigo.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleParser_Parse() {

	r := compiler.DirReader("./template/")
	p := compiler.New(r, nil, false)

	tree, err := p.Parse("index.html", ast.ContextHTML)
	if err != nil {
		log.Printf("parsing error: %s\n", err)
	}

	err = scrigo.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleMapReader() {

	r := compiler.MapReader(map[string][]byte{
		"header.csv": []byte("Name"),
		"names.csv":  []byte("{% include `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
	})

	src, _ := r.Read("names.csv", ast.ContextText)

	tree, err := compiler.ParseSource(src, ast.ContextText, false)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

	err = scrigo.RenderTree(os.Stdout, tree, nil, false)
	if err != nil {
		log.Printf("rendering error: %s\n", err)
	}

}

func ExampleDirReader() {

	// Creates a reader that read sources from the files in the directory
	// "template/".

	r := compiler.DirReader("template/")

	_, err := r.Read("index.html", ast.ContextHTML)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

}

func ExampleNewDirLimitedReader() {

	// Creates a reader that read sources from the files in the directory
	// "template/" and returns the error ErrReadTooLarge if a file to read
	// is larger than 50 KB or if all the read files exceed 256 KB.

	r := compiler.NewDirLimitedReader("template/", 50*1024, 256*1204)

	_, err := r.Read("very-large-file.csv", ast.ContextText)
	if err != nil {
		log.Printf("error: %s\n", err)
	}

}
