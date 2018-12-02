// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package template_test

import (
	"log"
	"os"

	"open2b/template"
)

func ExampleRender() {

	type Product struct {
		Name  string
		Price float32
	}

	src := []byte(`
  {% for i, p in products %}
  {{ i }}. {{ p.Name }}: $ {{ p.Price }}
  {% end %}`)

	vars := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	err := template.Render(os.Stdout, src, template.ContextText, vars)
	if err != nil {
		log.Printf("errors: %s\n", err)
	}

}

func ExampleRenderString() {

	type Product struct {
		Name  string
		Price float32
	}

	src := `
  {% for i, p in products %}
  {{ i }}. {{ p.Name }}: $ {{ p.Price }}
  {% end %}`

	vars := map[string]interface{}{
		"products": []Product{
			{Name: "Shirt", Price: 12.99},
			{Name: "Jacket", Price: 37.49},
		},
	}

	err := template.RenderString(os.Stdout, src, template.ContextText, vars)
	if err != nil {
		log.Printf("errors: %s\n", err)
	}

}

func ExampleDir_Render() {

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

	d := template.NewDir("./template/", template.ContextHTML)

	err := d.Render(os.Stderr, "index.html", vars)
	if err != nil {
		log.Printf("errors: %s\n", err)
	}

}

func ExampleMap_Render() {

	sources := map[string][]byte{
		"header.csv": []byte("Name"),
		"names.csv":  []byte("{% show `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
	}

	vars := map[string]interface{}{
		"names": []string{"Robert", "Mary", "Karen", "William", "Michelle"},
	}

	m := template.NewMap(sources, template.ContextText)

	err := m.Render(os.Stderr, "names.csv", vars)
	if err != nil {
		log.Printf("errors: %s\n", err)
	}

}
