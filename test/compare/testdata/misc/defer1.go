// run

// Test the defer statement.

// What is deferred?                             Package function
// Does the deferred function take arguments?    Yes/No
// Is the deferred function variadic?            Yes/No
// Are there more than one sibling defer?        Yes
// Are there more than one nested defer?         No

package main

import "fmt"

func f1() { fmt.Println("f1") }

func f2(i int) { fmt.Printf("f2 %d\n", i) }

func f3(f float64) { fmt.Printf("f3 %f\n", f) }

func f4(s string) { fmt.Printf("f4 %q\n", s) }

func f5(s []int) { fmt.Printf("f5 %v\n", s) }

func f6(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }

func f7(a int, b float64, c string) { fmt.Printf("f7 %d %f %q\n", a, b, c) }

func f8(a ...int) { fmt.Printf("f8 %v\n", a) }

func f9(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }

func f10() int { fmt.Println("f10"); return 5 }

func f11(a string, b float64, c []string) (string, error) {
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
	defer f11("a", 6.9, []string{"x", "y"})
	s := h("x", "y", "z")
	fmt.Printf("h %v\n", s)
	return a, "sf", 34.89, 8
}

func h(s ...string) []string {
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
