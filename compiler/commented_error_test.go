// Copyright (c) 2020 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"errors"
	"testing"
)

func Test_commentedError(t *testing.T) {
	cases := []struct {
		input          commentedError
		wantCSS        string
		wantHTML       string
		wantJavaScript string
	}{
		{
			input:          commentedError{errors.New("an error occurred")},
			wantCSS:        "/* an error occurred */",
			wantHTML:       "<!-- an error occurred -->",
			wantJavaScript: "/* an error occurred */",
		},

		{
			input:          commentedError{nil},
			wantCSS:        "",
			wantHTML:       "",
			wantJavaScript: "",
		},

		{
			input:          commentedError{errors.New("*int returned an error")},
			wantCSS:        "/* *int returned an error */",
			wantHTML:       "<!-- *int returned an error -->",
			wantJavaScript: "/* *int returned an error */",
		},
		{
			input:          commentedError{errors.New("bad */ error")},
			wantCSS:        "/* bad * / error */",
			wantHTML:       "<!-- bad */ error -->",
			wantJavaScript: "/* bad * / error */",
		},
		{
			input:          commentedError{errors.New("bad --> error")},
			wantCSS:        "/* bad --> error */",
			wantHTML:       "<!-- bad -- > error -->",
			wantJavaScript: "/* bad --> error */",
		},
		{
			input:          commentedError{errors.New("invalid char: \xc5; end")},
			wantCSS:        "/* invalid char: �; end */",
			wantHTML:       "<!-- invalid char: �; end -->",
			wantJavaScript: "/* invalid char: �; end */",
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {

			// CSS.
			gotCSS := cas.input.CSS()
			if gotCSS != cas.wantCSS {
				t.Errorf("CSS: got %q, want %q", gotCSS, cas.wantCSS)
			}

			// HTML.
			gotHTML := cas.input.HTML()
			if gotHTML != cas.wantHTML {
				t.Errorf("HTML: got %q, want %q", gotHTML, cas.wantHTML)
			}

			// JavaScript.
			gotJavaScript := cas.input.JavaScript()
			if gotJavaScript != cas.wantJavaScript {
				t.Errorf("JavaScript: got %q, want %q", gotJavaScript, cas.wantJavaScript)
			}

			// Plain text.
			gotPlainText := cas.input.String()
			if gotPlainText != "" {
				t.Errorf("Plain text: got %q, want \"\"", gotPlainText)
			}

		})
	}
}
