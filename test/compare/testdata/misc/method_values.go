// run

package main

import (
	. "github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {

	tv := Tv(2)
	tvd := &tv
	tp := Tp(3)
	tpd := &tp

	m1 := tv.Mv
	m2 := (&tv).Mv
	m3 := tvd.Mv
	m4 := (*tvd).Mv
	m5 := tp.Mp // tp.Mp is equivalent to (&tp).Mp
	m6 := (&tp).Mp
	m7 := tpd.Mp
	m8 := (*tpd).Mp
	m9 := ((*Tp)(nil)).Mp

	println(m1(1), m2(2), m3(3), m4(4), m5(5), m6(6), m7(7), m8(8), m9(9))

}
