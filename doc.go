// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template implements a template engine for Go for text, HTML, CSS
// and JavaScript files.
//
//
//  {% import conv "converter.html" %}
//  <html>
//  <body>
//      {% show "header.html" %}
//      <ul>
//          {% for p in products %}
//          <li>{{ p.Name }}: € {{ conv.Convert(p.Price, "EUR") }}</li>
//          {% end for %}
//      </ul>
//  </body>
//  </html>
//
// Functions Render and RenderString render a template source in a []byte or
// string value respectively:
//
//  src := []byte("{% for i, p in products %}{{ i }}. {{ p.name }}{% end %}")
//  err = template.Render(w, src, template.ContextText, vars)
//
// Method Render of Dir renders template sources in files located in a directory:
//
//  d := template.NewDir("./template/", template.ContextHTML)
//  err := d.Render(w, "index.html", vars)
//
// Method Render of Map renders template sources in map values with keys as
// paths:
//
//  sources := map[string][]byte{
//      "header.csv": []byte("Name"),
//      "names.csv":  []byte("{% show `header.csv` %}\n{% for name in names %}{{ name }}\n{% end %}"),
//  }
//  m := template.NewMap(sources, template.ContextText)
//  err := m.Render(w, "names.csv", vars)
//
// Advanced Usage
//
// Package template is for a basic usage. For an advanced usage see instead
// the sub-packages ast, parser and renderer.
//
//  // Advanced usage example.
//
//  import (
//      "open2b/template/ast"
//      "open2b/template/parser"
//      "open2b/template/renderer"
//  )
//
//  // Creates a reader to read from a directory with control of files size.
//  maxSize := 1024 * 1204
//  reader := parser.NewDirLimitedReader("./template/", maxSize, maxSize * 10)
//
//  // Creates a parses that read the files from the reader.
//  p := parser.New(reader)
//
//  // Parses a file and get the resulted tree expanded with extended
//  // and included files.
//  tree, err := p.Parse("index.html", ast.ContextHTML)
//  if err != nil {
//      log.Fatalf("parsing error: %s", err)
//  }
//
//  // Renders the parsed tree.
//  err = renderer.Render(w, tree, vars, func(err error) bool {
//      log.Printf("rendering error: %s\n", err)
//      return true
//  })
//
package template
