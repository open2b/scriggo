// Copyright 2021 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package builtin

import (
	"github.com/open2b/scriggo/native"
)

// Unsafeconv provides a package that contains functions to make unsafe conversions between string values and native
// types.
//
// Use these build options to use it in a template
//    opts := &scriggo.BuildOptions{
//        Globals: map[string]native.Declaration{
//            "unsafeconv": builtin.Unsafeconv,
//        },
//    }
//    template, err := scriggo.BuildTemplate(fsys, file, opts)
var Unsafeconv = native.Package{
	Name: "unsafeconv",
	Declarations: map[string]native.Declaration{
		"ToHTML": func(s string) native.HTML {
			return native.HTML(s)
		},
		"ToCSS": func(s string) native.CSS {
			return native.CSS(s)
		},
		"ToJS": func(s string) native.JS {
			return native.JS(s)
		},
		"ToJSON": func(s string) native.JSON {
			return native.JSON(s)
		},
		"ToMarkdown": func(s string) native.Markdown {
			return native.Markdown(s)
		},
	},
}
