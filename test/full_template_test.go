// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"bytes"
	"scriggo/template"
	"testing"
)

func TestFullTemplate(t *testing.T) {
	r := template.DirReader("./full_template")
	templ, err := template.Load("/index.html", r, nil, template.ContextHTML, nil)
	if err != nil {
		t.Fatal(err)
	}
	out := &bytes.Buffer{}
	err = templ.Render(out, nil, template.RenderOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != expectedOutput {
		if testing.Verbose() {
			t.Fatalf("expecting:\n%q\ngot:\n%q", expectedOutput, out.String())
		} else {
			t.Fatalf("output is not what expected (use -v to get details)")
		}
	}
}

const expectedOutput = `
<!DOCTYPE html>
`
