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
	defer testpkg.F1()
	defer testpkg.F2(1)
	defer testpkg.F3(1.2)
	defer testpkg.F4("a")
	defer testpkg.F5([]int{1, 2, 3})
	defer testpkg.F6(1, 3)
	defer testpkg.F7(5, 6.7, "c")
	defer testpkg.F8(3, 4, 5)
	defer testpkg.F9("a")
	defer testpkg.F9("a", 6)
	defer testpkg.F9("a", 7, 8, 9)
	defer testpkg.F9("a", []int{12, 16, 23}...)
	defer testpkg.F10()
	defer testpkg.F11("a", 6.9, []string{"x", "y"})
	s := h("x", "y", "z")
	fmt.Printf("h %v\n", s)
	return a, "sf", 34.89, 8
}

func h(s ...string) []string {
	defer testpkg.F11("a", 6.9, []string{"x", "y"})
	defer testpkg.F10()
	defer testpkg.F9("a", []int{12, 16, 23}...)
	defer testpkg.F9("a", 7, 8, 9)
	defer testpkg.F9("a", 6)
	defer testpkg.F9("a")
	defer testpkg.F8(3, 4, 5)
	defer testpkg.F7(5, 6.7, "c")
	defer testpkg.F6(1, 3)
	defer testpkg.F5([]int{1, 2, 3})
	defer testpkg.F4("a")
	defer testpkg.F3(1.2)
	defer testpkg.F2(1)
	defer testpkg.F1()
	return s
}
