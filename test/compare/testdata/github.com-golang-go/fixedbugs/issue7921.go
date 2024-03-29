// skip : '+build !gcflags_noopt' directive not supported and 'errorcheck -0 -m' mode not supported https://github.com/open2b/scriggo/issues/417

// +build !gcflags_noopt
// errorcheck -0 -m

// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package foo

import "bytes"

// In order to get desired results, we need a combination of
// both escape analysis and inlining.

func bufferNotEscape() string {
	// b itself does not escape, only its buf field will be
	// copied during String() call, but object "handle" itself
	// can be stack-allocated.
	var b bytes.Buffer
	b.WriteString("123")
	b.Write([]byte{'4'}) // ERROR "bufferNotEscape \[\]byte{...} does not escape$"
	return b.String()    // ERROR "inlining call to bytes.\(\*Buffer\).String$" "string\(bytes.b.buf\[bytes.b.off:\]\) escapes to heap$"
}

func bufferNoEscape2(xs []string) int { // ERROR "bufferNoEscape2 xs does not escape$"
	b := bytes.NewBuffer(make([]byte, 0, 64)) // ERROR "bufferNoEscape2 &bytes.Buffer{...} does not escape$" "bufferNoEscape2 make\(\[\]byte, 0, 64\) does not escape$" "inlining call to bytes.NewBuffer$"
	for _, x := range xs {
		b.WriteString(x)
	}
	return b.Len() // ERROR "inlining call to bytes.\(\*Buffer\).Len$"
}

func bufferNoEscape3(xs []string) string { // ERROR "bufferNoEscape3 xs does not escape$"
	b := bytes.NewBuffer(make([]byte, 0, 64)) // ERROR "bufferNoEscape3 &bytes.Buffer{...} does not escape$" "bufferNoEscape3 make\(\[\]byte, 0, 64\) does not escape$" "inlining call to bytes.NewBuffer$"
	for _, x := range xs {
		b.WriteString(x)
		b.WriteByte(',')
	}
	return b.String() // ERROR "inlining call to bytes.\(\*Buffer\).String$" "string\(bytes.b.buf\[bytes.b.off:\]\) escapes to heap$"
}

func bufferNoEscape4() []byte {
	var b bytes.Buffer
	b.Grow(64)       // ERROR "bufferNoEscape4 ignoring self-assignment in bytes.b.buf = bytes.b.buf\[:bytes.m·3\]$" "inlining call to bytes.\(\*Buffer\).Grow$"
	useBuffer(&b)
	return b.Bytes() // ERROR "inlining call to bytes.\(\*Buffer\).Bytes$"
}

func bufferNoEscape5() { // ERROR "can inline bufferNoEscape5$"
	b := bytes.NewBuffer(make([]byte, 0, 128)) // ERROR "bufferNoEscape5 &bytes.Buffer literal does not escape$" "bufferNoEscape5 make\(\[\]byte, 0, 128\) does not escape$" "inlining call to bytes.NewBuffer$"
	useBuffer(b)
}

//go:noinline
func useBuffer(b *bytes.Buffer) { // ERROR "useBuffer b does not escape$"
	b.WriteString("1234")
}
