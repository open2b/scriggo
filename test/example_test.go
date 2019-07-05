//+build ignore

// TODO(Gianluca): this file has been excluded from building. Consider what to
// modify and what to remove.

// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"log"
	"os"

	"scriggo"
	"scriggo/internal/compiler"
	"scriggo/ast"
)

func ExampleRenderSource() {
	type Product struct {
		Name  string
		Price float32
	}

	globals := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	src := `
 {% for i, p in products %}
 {{ i }}. {{ p.Name }}: $ {{ p.Price }}
 {% end %}`

	err := scriggo.RenderSource(os.Stdout, []byte(src), globals, false, scriggo.ContextText)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}

func ExampleRenderTree() {
	p := compiler.New(compiler.DirReader("/home/salinger/book/"), nil, false)

	tree, err := p.Parse("cover.html", ast.ContextHTML)
	if err != nil {
		log.Fatalf("parsing error: %s", err)
	}

	globals := map[string]interface{}{"title": "The Catcher in the Rye"}

	err = scriggo.RenderTree(os.Stdout, tree, globals, false)
	if err != nil {
		log.Fatalf("rendering error: %s", err)
	}
}

func ExampleDirRenderer() {
	type Product struct {
		Name  string
		Price float32
	}

	globals := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	r := scriggo.NewDirRenderer("./template/", false, scriggo.ContextHTML, false)

	err := r.Render(os.Stderr, "index.html", globals)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}

func ExampleMapRenderer() {
	sources := map[string][]byte{
		"header.csv": []byte("Name"),
		"names.csv":  []byte("{% include `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
	}

	vars := map[string]interface{}{
		"names": []string{"Robert", "Mary", "Karen", "William", "Michelle"},
	}

	r := scriggo.NewMapRenderer(sources, false, scriggo.ContextText)

	err := r.Render(os.Stderr, "names.csv", vars)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}
