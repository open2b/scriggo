// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package template implements a template engine for Go for text, HTML, CSS
// and JavaScript files.
//
//  {% import conv "converter.html" %}
//  <html>
//  <body>
//      {% include "header.html" %}
//      <ul>
//          {% for p in products %}
//          <li>{{ p.Name }}: € {{ conv.Convert(p.Price, "EUR") }}</li>
//          {% end for %}
//      </ul>
//  </body>
//  </html>
//
// Common usage examples
//
// Render template files:
//
//  // Creates a renderer that reads template sources from a directory.
//  r := template.NewDirRenderer("template/", false, template.ContextHTML)
//
//  // Renders a file with variables vars.
//  err := r.Render(os.Stdout, "page.html", vars)
//
// Render a source:
//
//  err := template.RenderSource(os.Stdout, []byte(`{{ a + b }}`), vars)
//
// Advanced usage examples
//
// Get a template tree from files:
//
//  import (
//      "open2b/template/ast"
//      "open2b/template/parser"
//  )
//
//  // Creates a reader that read the sources from a directory.
//  r := parser.DirReader("template/")
//
//  // Creates a parser.
//  p := parser.New(r)
//
//  // Parses a path and gets the corresponding tree.
//  tree, err := p.Parse("page.html", ast.ContextHTML)
//
// Gets a tree from a source:
//
//  import "open2b/template/parser"
//
//  tree, err := parser.ParseSource([]byte(`{{ a + b }}`), ast.ContextText)
//
// Parse, transform and render a tree:
//
//  import (
//      "open2b/template/ast"
//      "open2b/template/parser"
//  )
//
//  // Defines the transformation function.
//  transform := func(tree *ast.Tree) (*ast.Tree, err) {
//      ...
//  }
//
//  // Creates a transformer that reads the files from a directory.
//  tr := parser.NewTransformerReader(parser.DirReader("template/"), transform)
//
//  // Creates a parser.
//  p := parser.New(tr)
//
//  // Parses the files transforming the resulting tree.
//  tree, err := p.Parse("page.html", ast.ContextHTML)
//
//  // Renders the tree.
//  err := template.RenderTree(os.Stdout, tree, vars)
//
// Read sources from files limiting sizes:
//
//  import "open2b/template/parser"
//
//  // Creates a reader that limit file sizes to 50K and total bytes read to 1M.
//  r := parser.NewDirLimitedReader("template/", 50 * 1024, 1024 * 1024)
//
// Variables
//
// Variables, available during the rendering of the template, are defined by
// the vars parameter of a rendering function or method. vars can be:
//
//   * nil
//   * a map with a key of type string
//   * a type with underlying type a map with a key of type string
//   * a struct or pointer to struct
//   * a reflect.Value whose concrete value meets one of the previous ones
//
// If vars is a map or have type with underlying type a map, the map keys that
// are valid identifier names in Go and their values will be names and values
// of variables.
//
// If vars is a struct or pointer to a struct, the names of the exported
// fields of the struct will be names and values of the global variables.
// If an exported field has the tag "template" the variable name is defined
// by the tag.
//
// For example, if vars has type:
//
//  struct {
//      Name          string `template:"name"`
//      StockQuantity int    `template:"stock"`
//  }
//
// The variables will be "name" and "stock".
//
// If vars is nil, there will be no global variables besides the builtin
// variables.
//
// Functions
//
// To define a function in the template, assign a function value to a
// variable:
//
//  vars := map[string]interface{}{
//      "f" : func (a int, b string) (bool, error) { ... },
//  }
//
// A function can have any number of parameters, can be variadic and must have
// one or two results. If it has two results, the second must have a type
// error. Only the first result is returned and the error is treated as an
// execution error in an expression.
//
//  {% var r = f(5, "abc") %}
//
// A function must not modify its arguments, which means it must not modify
// slice elements, struct field values, and map keys and values.
//
// Types
//
// Each template type is implemented with a type of Go. The following are the
// template types and their implementation types in Go:
//
//  bool:     the bool type
//
//  string:   the string type and the types implementing the interface
//            renderer.Stringer
//
//  html:     the type template.HTML
//
//  number:   all integer and floating-point types (excluding uintptr),
//            *apd.Decimal [ https://github.com/cockroachdb/apd ] and the
//            types implementing the interface renderer.Numberer
//
//  int:      the int type
//
//  rune:     the int32 (rune) type
//
//  byte:     the uint8 (byte) type
//
//  map:      the type template.Map, a struct type, a pointer to struct type
//            and a map type
//
//  slice:    the type template.Slice and a slice type
//
//  bytes:    the type template.Bytes and the []byte type
//
//  function: a function type with only one return value. As numeric parameter
//            types, only int and *apd.Decimal can be used
//
// If a value has a type that implements Renderer, the method Render will be
// called when the value have to be rendered. See the Renderer documentation.
//
// If a value has a map type, their keys will be the names of the fields of
// the template struct.
//
// If a value has type struct or pointer to a struct, the names of the
// exported fields of the struct will be the field names of the template
// struct. If an exported field has the tag "template" the field name is
// defined by the tag as for the vars.
//
package template
