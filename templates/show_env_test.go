// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/open2b/scriggo/runtime"

	"github.com/google/go-cmp/cmp"
)

// EnvStringer.

type testEnvStringer struct{}

func (*testEnvStringer) String(env runtime.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testEnvStringerValue = &testEnvStringer{}

// HTMLEnvStringer.

type testHTMLEnvStringer struct{}

func (*testHTMLEnvStringer) HTML(env runtime.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testHTMLEnvStringerValue = &testHTMLEnvStringer{}

// CSSEnvStringer.

type testCSSEnvStringer struct{}

func (*testCSSEnvStringer) CSS(env runtime.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testCSSEnvStringerValue = &testCSSEnvStringer{}

// JSEnvStringer.

type testJSEnvStringer struct{}

func (*testJSEnvStringer) JS(env runtime.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testJSEnvStringerValue = &testJSEnvStringer{}

// JSONEnvStringer.

type testJSONEnvStringer struct{}

func (*testJSONEnvStringer) JSON(env runtime.Env) string {
	return fmt.Sprint(env.Context().Value("forty-two"))
}

var testJSONEnvStringerValue = &testJSONEnvStringer{}

// ---

var envStringerCases = map[string]struct {
	sources map[string]string
	globals map[string]interface{}
	format  Format
	want    string
}{
	"EnvStringer": {
		sources: map[string]string{
			"index.txt": "value read from env is {{ v }}",
		},
		globals: map[string]interface{}{
			"v": &testEnvStringerValue,
		},
		format: FormatText,
		want:   "value read from env is 42",
	},
	"HTMLEnvStringer": {
		sources: map[string]string{
			"index.html": "value read from env is {{ v }}",
		},
		globals: map[string]interface{}{
			"v": &testHTMLEnvStringerValue,
		},
		format: FormatHTML,
		want:   "value read from env is 42",
	},
	"CSSEnvStringer": {
		sources: map[string]string{
			"index.css": "border-radius: {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testCSSEnvStringerValue,
		},
		format: FormatCSS,
		want:   "border-radius: 42;",
	},
	"JSEnvStringer": {
		sources: map[string]string{
			"index.js": "var x = {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testJSEnvStringerValue,
		},
		format: FormatJS,
		want:   "var x = 42;",
	},
	"JSONEnvStringer": {
		sources: map[string]string{
			"index.json": "var x = {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testJSONEnvStringerValue,
		},
		format: FormatJSON,
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
			fsys := MapFS{}
			for p, src := range cas.sources {
				fsys[p] = src
			}
			opts := &BuildOptions{
				Globals: cas.globals,
			}
			name := "index.txt"
			switch cas.format {
			case FormatHTML:
				name = "index.html"
			case FormatCSS:
				name = "index.css"
			case FormatJS:
				name = "index.js"
			case FormatJSON:
				name = "index.json"
			}
			template, err := Build(fsys, name, opts)
			if err != nil {
				t.Fatal(err)
			}
			w := &bytes.Buffer{}
			options := &RunOptions{Context: ctx}
			err = template.Run(w, nil, options)
			if diff := cmp.Diff(cas.want, w.String()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}
