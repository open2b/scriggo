// run

package main

import "fmt"

var f1 = func() { fmt.Println("f1") }

var f2 = func(i int) { fmt.Printf("f2 %d\n", i) }

var f3 = func(f float64) { fmt.Printf("f3 %f\n", f) }

var f4 = func(s string) { fmt.Printf("f4 %q\n", s) }

var f5 = func(s []int) { fmt.Printf("f5 %v\n", s) }

var f6 = func(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }

var f7 = func(a int, b float64, c string) { fmt.Printf("f7 %d %f %q\n", a, b, c) }

var f8 = func(a ...int) { fmt.Printf("f8 %v\n", a) }

var f9 = func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }

var f10 = func() int { fmt.Println("f10"); return 5 }

var f11 = func(a string, b float64, c []string) (string, error) {
	fmt.Printf("f11 %q %f %v\n", a, b, c)
	return "", nil
}

func main() {
	a, b, c, d := g()
	fmt.Println(a, b, c, d)
}

func g() (int, string, float64, int) {
	var a = 5
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
	//defer f11("a", 6.9, []string{"x", "y"})
	s := h("x", "y", "z")
	fmt.Printf("h %v\n", s)
	return a, "sf", 34.89, 8
}

func h(s ...string) []string {
	//defer f11("a", 6.9, []string{"x", "y"})
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
