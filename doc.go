// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template implements a template engine for Go for text, HTML, CSS
// and JavaScript files.
//
//
//    {% import conv "converter.html" %}
//    <html>
//    <body>
//      {% show "header.html" %}
//      <ul>
//        {% for p in products %}
//        <li>{{ p.Name }}: € {{ conv.Convert(p.Price, "EUR") }}</li>
//        {% end for %}
//      </ul>
//    </body>
//    </html>`
//
// Function RenderString renders a template source in a string:
//
//   src := "{% for i, p in products %}{{ i }}. {{ p.name }}{% end %}"
//   err = template.RenderString(w, src, template.ContextText, vars)
//
// Method Render renders a template with files located in a directory:
//
//   t := template.New("./template/", template.ContextHTML)
//   err := t.Render(w, "/index.html", vars)
//
// Advanced functionalities
//
// Package template implements only basic functionalities. For more
// control on how to read template files, build template trees and render
// a tree use instead the sub-packages ast, parser and renderer.
//
//   import (
//       "open2b/template/ast"
//       "open2b/template/parser"
//       "open2b/template/renderer"
//   )
//
//   // Creates a reader to read from a directory with control of files size.
//   maxSize := 1024 * 1204
//   reader := NewDirLimitedReader("./template/", maxSize, maxSize * 10)
//
//   // Creates a parses that read the files from the reader.
//   p := parser.New(reader)
//
//   // Parses a file and get the resulted tree expanded with extended
//   // and included files.
//   tree, err := p.Parse("/index.html", ast.ContextHTML)
//   if err != nil {
//       log.Fatalf("parsing error: %s", err)
//   }
//
//   // Renders the tree.
//   err = renderer.Render(w, tree, vars, func(err error) bool {
//       log.Printf("rendering error: %s\n", err)
//       returns true
//   })
//
package template
