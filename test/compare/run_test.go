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
				`	x := 3 // ERROR "x declared and not used"`,
				`}`,
			}),
			ext: ".go",
		},
	}

	for _, test := range mustPass {
		errorcheck([]byte(test.src), test.ext, nil)
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
			panic: "error does not match:\n\n\texpecting:  undefined: A\n\tgot:        main:3:6: undefined: a",
		},
	}

	for _, test := range mustFail {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("should panic!")
				}
				got := r.(error).Error()
				if test.panic != got {
					t.Fatalf("expecting %q, got %q", test.panic, got)
				}
			}()
			errorcheck([]byte(test.src), test.ext, nil)
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
			ext:          `.sgo`,
			expectedMode: `compile`,
			expectedArgs: []string{"-time=3s"},
		},
		{
			src:          `// compile -time = 3s`,
			ext:          `.sgo`,
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
			mode, args := readMode([]byte(c.src), c.ext)
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
