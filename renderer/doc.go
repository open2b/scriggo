// Copyright (c) 2018 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package renderer implements methods to render template trees.
//
// To render a tree use the function Render, where w is the writer where to
// write the result and vars declares the global variables that will be
// defined during rendering:
//
//  err := renderer.Render(w, path, vars, h)
//
// Global Variables
//
// Global variables are defined by the vars parameter of Render. vars can be:
//
//   * nil
//   * a map with a key of type string
//   * a type with underlying type one of the previous map types
//   * a struct or pointer to struct
//   * a reflect.Value whose concrete value meets one of the previous ones
//
// If vars is a map or have type with underlying type a map, the map keys that
// are valid identifier names in Go and their values will be names and values
// of the global variables.
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
// The global variables will be "name" and "stock".
//
// If vars is nil, there will be no global variables besides the builtin
// variables.
//
// Types
//
// Each template type is implemented with a type of Go. The following are the
// template types and their implementation types in Go:
//
//  bool:     the bool type
//
//  string:   the types string, renderer.HTML and the types implementing the
//            interface renderer.Stringer
//
//  number:   all integer and floating-point types (excluding uintptr),
//            decimal.Decimal [github.com/shopspring/decimal] and the types
//            implementing the interface renderer.Numberer
//
//  struct:   a struct pointer type, a map type with keys of type string
//            and the types convertible to these types
//
//  slice:    a slice type
//
//  function: a function type with only one return value. As numeric parameter
//            types, only int and decimal.Decimal can be used.
//
// If a value has a type that implements renderer.WriterTo, the method WriteTo
// will be called when the value have to be written to the writer w. See the
// documentation of renderer.WriterTo.
//
// If a value has a map type, their keys will be the names of the fields of the
// template struct.
//
// If a value has type pointer to a struct, the names of the exported fields
// of the struct will be the field names of the template struct. If an exported
// field has the tag "template" the field name is defined by the tag as for the
// vars.
//
package renderer
