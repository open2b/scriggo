// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "testing"

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
		errorcheck([]byte(test.src), test.ext)
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
			errorcheck([]byte(test.src), test.ext)
		}()
	}

}
