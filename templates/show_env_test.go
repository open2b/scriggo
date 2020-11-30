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
	sources  map[string]string
	globals  map[string]interface{}
	language Language
	want     string
}{
	"EnvStringer": {
		sources: map[string]string{
			"index.html": "value read from env is {{ v }}",
		},
		globals: map[string]interface{}{
			"v": &testEnvStringerValue,
		},
		language: LanguageText,
		want:     "value read from env is 42",
	},
	"HTMLEnvStringer": {
		sources: map[string]string{
			"index.html": "value read from env is {{ v }}",
		},
		globals: map[string]interface{}{
			"v": &testHTMLEnvStringerValue,
		},
		language: LanguageHTML,
		want:     "value read from env is 42",
	},
	"CSSEnvStringer": {
		sources: map[string]string{
			"index.html": "border-radius: {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testCSSEnvStringerValue,
		},
		language: LanguageCSS,
		want:     "border-radius: 42;",
	},
	"JSEnvStringer": {
		sources: map[string]string{
			"index.html": "var x = {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testJSEnvStringerValue,
		},
		language: LanguageJS,
		want:     "var x = 42;",
	},
	"JSONEnvStringer": {
		sources: map[string]string{
			"index.html": "var x = {{ v }};",
		},
		globals: map[string]interface{}{
			"v": &testJSONEnvStringerValue,
		},
		language: LanguageJSON,
		want:     "var x = 42;",
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
			r := MapReader{}
			for p, src := range cas.sources {
				r[p] = []byte(src)
			}
			opts := &LoadOptions{
				Globals: cas.globals,
			}
			template, err := Load("index.html", r, cas.language, opts)
			if err != nil {
				t.Fatal(err)
			}
			w := &bytes.Buffer{}
			options := &RenderOptions{Context: ctx}
			err = template.Render(w, nil, options)
			if diff := cmp.Diff(cas.want, w.String()); diff != "" {
				t.Fatalf("(-want, +got):\n%s", diff)
			}
		})
	}
}
