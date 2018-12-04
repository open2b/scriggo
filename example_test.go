// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template_test

import (
	"log"
	"os"

	"open2b/template"
	"open2b/template/ast"
	"open2b/template/parser"
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

	err := template.RenderSource(os.Stdout, []byte(src), template.ContextText, vars, nil)
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

	err = template.RenderTree(os.Stdout, tree, vars, nil)
	if err != nil {
		log.Fatalf("rendering error: %s", err)
	}
}

func ExampleDirRenderer_Render() {
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

	r := template.NewDirRenderer("./template/", template.ContextHTML)

	err := r.Render(os.Stderr, "index.html", vars, nil)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}

func ExampleMapRenderer_Render() {
	sources := map[string][]byte{
		"header.csv": []byte("Name"),
		"names.csv":  []byte("{% show `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
	}

	vars := map[string]interface{}{
		"names": []string{"Robert", "Mary", "Karen", "William", "Michelle"},
	}

	r := template.NewMapRenderer(sources, template.ContextText)

	err := r.Render(os.Stderr, "names.csv", vars, nil)
	if err != nil {
		log.Printf("error: %s\n", err)
	}
}
