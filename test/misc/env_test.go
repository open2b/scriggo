// Copyright 2019 The Scriggo Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package misc

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/internal/fstest"
	"github.com/open2b/scriggo/native"

	"github.com/google/go-cmp/cmp"
)

// EnvStringer.

type testEnvStringer struct{}

func (*testEnvStringer) String(env native.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testEnvStringerValue = &testEnvStringer{}

// HTMLEnvStringer.

type testHTMLEnvStringer struct{}

func (*testHTMLEnvStringer) HTML(env native.Env) native.HTML {
	return native.HTML(fmt.Sprint(env.Context().Value("forty-two")))
}

var testHTMLEnvStringerValue = &testHTMLEnvStringer{}

// CSSEnvStringer.

type testCSSEnvStringer struct{}

func (*testCSSEnvStringer) CSS(env native.Env) native.CSS {
	return native.CSS(fmt.Sprint(env.Context().Value("forty-two")))
}

var testCSSEnvStringerValue = &testCSSEnvStringer{}

// JSEnvStringer.

type testJSEnvStringer struct{}

func (*testJSEnvStringer) JS(env native.Env) native.JS {
	return native.JS(fmt.Sprint(env.Context().Value("forty-two")))
}

var testJSEnvStringerValue = &testJSEnvStringer{}

// JSONEnvStringer.

type testJSONEnvStringer struct{}

func (*testJSONEnvStringer) JSON(env native.Env) native.JSON {
	return native.JSON(fmt.Sprint(env.Context().Value("forty-two")))
}

var testJSONEnvStringerValue = &testJSONEnvStringer{}

// ---

var envStringerCases = map[string]struct {
	sources map[string]string
	globals native.Declarations
	format  scriggo.Format
	want    string
}{
	"EnvStringer": {
		sources: map[string]string{
			"index.txt": "value read from env is {{ v }}",
		},
		globals: native.Declarations{
			"v": &testEnvStringerValue,
		},
		format: scriggo.FormatText,
		want:   "value read from env is 42",
	},
	"HTMLEnvStringer": {
		sources: map[string]string{
			"index.html": "value read from env is {{ v }}",
		},
		globals: native.Declarations{
			"v": &testHTMLEnvStringerValue,
		},
		format: scriggo.FormatHTML,
		want:   "value read from env is 42",
	},
	"CSSEnvStringer": {
		sources: map[string]string{
			"index.css": "border-radius: {{ v }};",
		},
		globals: native.Declarations{
			"v": &testCSSEnvStringerValue,
		},
		format: scriggo.FormatCSS,
		want:   "border-radius: 42;",
	},
	"JSEnvStringer": {
		sources: map[string]string{
			"index.js": "var x = {{ v }};",
		},
		globals: native.Declarations{
			"v": &testJSEnvStringerValue,
		},
		format: scriggo.FormatJS,
		want:   "var x = 42;",
	},
	"JSONEnvStringer": {
		sources: map[string]string{
			"index.json": "var x = {{ v }};",
		},
		globals: native.Declarations{
			"v": &testJSONEnvStringerValue,
		},
		format: scriggo.FormatJSON,
		want:   "var x = 42;",
	},
}

// TestEnvStringer tests these interfaces:
//
//  * EnvStringer
//  * HTMLEnvStringer
//  * CSSEnvStringer
//  * JSEnvStringer
//  * JSONEnvStringer
//
func TestEnvStringer(t *testing.T) {
	for name, cas := range envStringerCases {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), "forty-two", 42)
			fsys := fstest.Files{}
			for p, src := range cas.sources {
				fsys[p] = src
			}
			opts := &scriggo.BuildOptions{
				Globals: cas.globals,
			}
			name := "index.txt"
			switch cas.format {
			case scriggo.FormatHTML:
				name = "index.html"
			case scriggo.FormatCSS:
				name = "index.css"
			case scriggo.FormatJS:
				name = "index.js"
			case scriggo.FormatJSON:
				name = "index.json"
			}
			template, err := scriggo.BuildTemplate(fsys, name, opts)
			if err != nil {
				t.Fatal(err)
			}
			w := &bytes.Buffer{}
			options := &scriggo.RunOptions{Context: ctx}
			err = template.Run(w, nil, options)
			if diff := cmp.Diff(cas.want, w.String()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}
