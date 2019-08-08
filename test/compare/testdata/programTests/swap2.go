// run

package main

import "testpkg"

func swap(a, b int) (int, int) {
	return b, a
}

func sum(a, b, c int) int {
	s := 0
	s += a
	s += b
	s += c
	return s
}

func fact(n int) int {
	switch n {
	case 0:
		return 1
	case 1:
		return 1
	default:
		return n * fact(n-1)
	}
}

func main() {
	var a, b, c, d, e, f int

	a = 2
	b = 6
	a, b = swap(a, b)
	c, d = swap(a, b)
	e = sum(a, b, sum(c, d, 0))
	f = fact(d+2) + sum(a, b, c)

	_ = a
	_ = b
	_ = c
	_ = d
	_ = e
	_ = f

	testpkg.PrintString("a:")
	testpkg.PrintInt(a)
	testpkg.PrintString(",")
	testpkg.PrintString("b:")
	testpkg.PrintInt(b)
	testpkg.PrintString(",")
	testpkg.PrintString("c:")
	testpkg.PrintInt(c)
	testpkg.PrintString(",")
	testpkg.PrintString("d:")
	testpkg.PrintInt(d)
	testpkg.PrintString(",")
	testpkg.PrintString("e:")
	testpkg.PrintInt(e)
	testpkg.PrintString(",")
	testpkg.PrintString("f:")
	testpkg.PrintInt(f)
	testpkg.PrintString(",")

	return
}
