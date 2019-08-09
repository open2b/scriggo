// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	a, b, c, d := g()
	fmt.Println(a, b, c, d)
}

func g() (int, string, float64, int) {
	var a = 5

	f1 := testpkg.F1
	f2 := testpkg.F2
	f3 := testpkg.F3
	f4 := testpkg.F4
	f5 := testpkg.F5
	f6 := testpkg.F6
	f7 := testpkg.F7
	f8 := testpkg.F8
	f9 := testpkg.F9
	f10 := testpkg.F10
	f11 := testpkg.F11

	defer f1()
	defer f2(1)
	defer f3(1.2)
	defer f4("a")
	defer f5([]int{1, 2, 3})
	defer f6(1, 3)
	defer f7(5, 6.7, "c")
	defer f8(3, 4, 5)
	defer f9("a")
	defer f9("a", 6)
	defer f9("a", 7, 8, 9)
	defer f9("a", []int{12, 16, 23}...)
	defer f10()
	defer f11("a", 6.9, []string{"x", "y"})

	s := h("x", "y", "z")
	fmt.Printf("h %v\n", s)

	return a, "sf", 34.89, 8
}

func h(s ...string) []string {

	f1 := testpkg.F1
	f2 := testpkg.F2
	f3 := testpkg.F3
	f4 := testpkg.F4
	f5 := testpkg.F5
	f6 := testpkg.F6
	f7 := testpkg.F7
	f8 := testpkg.F8
	f9 := testpkg.F9
	f10 := testpkg.F10
	f11 := testpkg.F11

	defer f11("a", 6.9, []string{"x", "y"})
	defer f10()
	defer f9("a", []int{12, 16, 23}...)
	defer f9("a", 7, 8, 9)
	defer f9("a", 6)
	defer f9("a")
	defer f8(3, 4, 5)
	defer f7(5, 6.7, "c")
	defer f6(1, 3)
	defer f5([]int{1, 2, 3})
	defer f4("a")
	defer f3(1.2)
	defer f2(1)
	defer f1()

	return s
}
