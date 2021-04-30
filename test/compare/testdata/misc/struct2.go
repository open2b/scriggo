// run

package main

import (
	"fmt"
	"testpkg"
)

type t int
type T []int

type S1 struct{ int }

type S2 struct{ *int }

type S3 struct{ t }

type S4 struct{ *t }

type S5 struct{ T }

type S6 struct{ *T }

type S7 struct{ testpkg.T }

type S8 struct{ *testpkg.T }

func main() {

	s1 := S1{1}
	if s1.int != 1 {
		panic(fmt.Sprintf("s1.int == %d", s1.int))
	}

	i2 := 2
	s2 := S2{&i2}
	if *s2.int != 2 {
		panic(fmt.Sprintf("s1.int == %d", *s2.int))
	}

	s3 := S3{3}
	if s3.t != 3 {
		panic(fmt.Sprintf("s3.t == %d", s3.t))
	}

	i4 := t(4)
	s4 := S4{&i4}
	if *s4.t != 4 {
		panic(fmt.Sprintf("s4.t == %d", *s4.t))
	}

	s5 := S5{[]int{5}}
	if s5.T[0] != 5 {
		panic(fmt.Sprintf("s5.T[0] == %d", s5.T[0]))
	}

	i6 := T([]int{5})
	s6 := S6{&i6}
	if (*s6.T)[0] != 5 {
		panic(fmt.Sprintf(" (*s6.T)[0] == %d", (*s6.T)[0]))
	}

	s7 := S7{7}
	if s7.T != 7 {
		panic(fmt.Sprintf("s7.T == %d", s7.T))
	}

	i8 := testpkg.T(8)
	s8 := S8{&i8}
	if *s8.T != 8 {
		panic(fmt.Sprintf("*s8.T == %d", *s8.T))
	}

}
