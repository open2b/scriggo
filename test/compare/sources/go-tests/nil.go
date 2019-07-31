

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "fmt"

func main() {
	var i *int
	var f *float32
	var s *string
	var m map[float32]*int
	var c chan int

	i = nil
	f = nil
	s = nil
	m = nil
	c = nil
	i = nil

	fmt.Print(i, f, s, m, c)

	arraytest()
	maptest()
	slicetest()
}

// nil array pointer

func arraytest() {
	// TODO(Gianluca).
	// var p *[10]int

	// // Looping over indices is fine.
	// s := 0
	// for i := range p {
	// 	s += i
	// }
	// if s != 45 {
	// 	panic(s)
	// }

	// s = 0
	// for i := 0; i < len(p); i++ {
	// 	s += i
	// }
	// if s != 45 {
	// 	panic(s)
	// }


	// // Looping over values is not.
	// shouldPanic(func() {
	// 	for i, v := range p {
	// 		s += i + v
	// 	}
	// })

	// shouldPanic(func() {
	// 	for i := 0; i < len(p); i++ {
	// 		s += p[i]
	// 	}
	// })
}


// nil map

func maptest() {
	var m map[int]int

	// nil map appears empty
	if len(m) != 0 {
		panic(len(m))
	}
	if m[1] != 0 {
		panic(m[1])
	}
	if x, ok := m[1]; x != 0 || ok {
		panic(fmt.Sprint(x, ok))
	}

	for k, v := range m {
		panic(k)
		panic(v)
	}

	fmt.Print(m)
	// can delete (non-existent) entries
	delete(m, 2)
	fmt.Print(m)
}

// nil slice

func slicetest() {
	var x []int

	// nil slice is just a 0-element slice.
	if len(x) != 0 {
		panic(len(x))
	}
	if cap(x) != 0 {
		panic(cap(x))
	}

	// no 0-element slices can be read from or written to
	var s int
	fmt.Print(s)
}
