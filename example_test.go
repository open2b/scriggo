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

func ExampleTemplate_Render() {

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

	t := template.New("template/", template.ContextHTML)

	err := t.Render(os.Stderr, "index.html", vars)
	if err != nil {
		log.Printf("errors: %s\n", err)
	}

}
