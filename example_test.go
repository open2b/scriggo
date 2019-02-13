// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scrigo_test

import (
	"log"
	"os"

	"scrigo"
	"scrigo/ast"
	"scrigo/parser"
)

func ExampleRenderSource() {
	type Product struct {
		Name  string
		Price float32
	}

	vars := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	src := `
 {% for i, p in products %}
 {{ i }}. {{ p.Name }}: $ {{ p.Price }}
 {% end %}`

	err := scrigo.RenderSource(os.Stdout, []byte(src), vars, false, scrigo.ContextText)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}

func ExampleRenderTree() {
	p := parser.New(parser.DirReader("/home/salinger/book/"))

	tree, err := p.Parse("cover.html", ast.ContextHTML)
	if err != nil {
		log.Fatalf("parsing error: %s", err)
	}

	vars := map[string]string{"title": "The Catcher in the Rye"}

	err = scrigo.RenderTree(os.Stdout, tree, vars, false)
	if err != nil {
		log.Fatalf("rendering error: %s", err)
	}
}

func ExampleDirRenderer() {
	type Product struct {
		Name  string
		Price float32
	}

	vars := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	r := scrigo.NewDirRenderer("./template/", false, scrigo.ContextHTML)

	err := r.Render(os.Stderr, "index.html", vars)
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

	r := scrigo.NewMapRenderer(sources, false, scrigo.ContextText)

	err := r.Render(os.Stderr, "names.csv", vars)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}
