// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"testing"
)

func Test_errorcheck(t *testing.T) {

	// Build the executable 'cmd/cmd', that is going to be used in this test.
	err := buildCmd()
	if err != nil {
		t.Fatalf("cannot build cmd: %s", err)
	}

	mustPass := []struct {
		src string
		ext string
	}{
		{
			src: joinLines([]string{
				`package main`,
				`func main() {`,
				`	_ = a // ERROR "undefined: a"`,
				`	_ = b // ERROR "undefined: b"`,
				`	_ = c // ERROR "undefined: c"`,
				`}`,
			}),
			ext: ".go",
		},
		{
			src: joinLines([]string{
				`package main`,
				`func main() {`,
				`	x := 3 // ERROR "x declared but not used"`,
				`}`,
			}),
			ext: ".go",
		},
	}

	for _, test := range mustPass {
		err := errorcheck([]byte(test.src), test.ext, nil)
		if err != nil {
			t.Fatalf("unexpected error %v", err)
		}
	}

	mustFail := []struct {
		src   string
		ext   string
		panic string
	}{
		{
			src: joinLines([]string{
				`package main`,
				`func main() {`,
				`	_ = a // ERROR "undefined: A"`,
				`}`,
			}),
			ext:   ".go",
			panic: "output does not match with the given regular expression:\n\n\tregexp:    undefined: A\n\toutput:    undefined: a\n",
		},
	}

	for _, test := range mustFail {
		func() {
			err := errorcheck([]byte(test.src), test.ext, nil)
			if err == nil {
				t.Fatal("should return a not nil error")
			}
			got := err.Error()
			if test.panic != got {
				t.Fatalf("expecting %q, got %q", test.panic, got)
			}
		}()
	}

}

func Test_readMode(t *testing.T) {
	cases := []struct {
		src          string
		ext          string
		expectedMode string
		expectedArgs []string
	}{
		{
			src:          `// run`,
			ext:          `.go`,
			expectedMode: `run`,
		},
		{
			src:          `// run -option`,
			ext:          `.go`,
			expectedMode: `run`,
			expectedArgs: []string{"-option"},
		},
		{
			src:          `// compile -time=3s`,
			ext:          `.go`,
			expectedMode: `compile`,
			expectedArgs: []string{"-time=3s"},
		},
		{
			src:          `// compile -time=3s`,
			ext:          `.script`,
			expectedMode: `compile`,
			expectedArgs: []string{"-time=3s"},
		},
		{
			src:          `// compile -time = 3s`,
			ext:          `.script`,
			expectedMode: `compile`,
			expectedArgs: []string{"-time", "=", "3s"},
		},
		{
			src: `{# render -time=5ms #}
text`,
			ext:          `.html`,
			expectedMode: `render`,
			expectedArgs: []string{"-time=5ms"},
		},
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			mode, args, err := readMode([]byte(c.src), c.ext)
			if err != nil {
				t.Errorf("expecting mode '%s', got error '%s'", c.expectedMode, err)
			}
			if mode != c.expectedMode {
				t.Errorf("expecting mode '%s', got '%s'", c.expectedMode, mode)
			}
			if len(args) == 0 && len(c.expectedArgs) == 0 {
				// Ok.
			} else if !reflect.DeepEqual(args, c.expectedArgs) {
				t.Errorf("expecting args %#v, got %#v", c.expectedArgs, args)
			}
		})
	}
}
