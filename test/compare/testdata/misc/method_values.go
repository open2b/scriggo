// run

package main

import (
	"fmt"

	. "github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {

	x1 := Mv(3)
	m1v := x1.Mv
	fmt.Printf("x1.Mv: %d\n", m1v(2))
	m1p := x1.Mp
	fmt.Printf("x1.Mp: %d\n", m1p(2))

	v := Mv(7)
	x2 := &v
	m2v := x2.Mv
	fmt.Printf("x2.Mv: %d\n", m2v(1))
	m2p := x2.Mp
	fmt.Printf("x2.Mp: %d\n", m2p(1))

	var x3 MiV = x1
	m3v := x3.Mv
	fmt.Printf("x3.Mv: %d\n", m3v(2))

	var x4 MiP = x2
	m4p := x4.Mp
	fmt.Printf("x4.Mp: %d\n", m4p(2))

	var x5 *Mv
	m5 := x5.MpNil
	fmt.Printf("x5.MpNil: %d\n", m5(2, 3))
}
