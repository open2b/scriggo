// run

package main

import "fmt"

func main() {
	a, b, c, d := f()
	fmt.Println(a, b, c, d)
}

func f() (int, string, float64, int) {
	var a = 5
	defer func() { fmt.Println("f1") }()
	defer func(i int) { fmt.Printf("f2 %d\n", i) }(981632)
	defer func(f float32) { fmt.Printf("f3 %f\n", f) }(9.640173)
	defer func(s string) { fmt.Printf("f4 %q\n", s) }("test123test")
	defer func(s []int) { fmt.Printf("f5 %v\n", s) }(nil)
	defer func(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }(9, 8)
	defer func(a []int, b string, c float32) { fmt.Printf("f7 %v %s %f\n", a, b, c) }([]int{0}, "", 23.09471)
	defer func(a ...int) { fmt.Printf("f8 %v\n", a) }(9, 8, 7, 6, 5, 4, 3, 2, 1)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("a")
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", 3)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", 6, 3, 0)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", []int{6, 3, 0}...)
	defer func() int { fmt.Println("f10"); return 5 }()
	defer func(a rune, b string, c int) (string, error) {
		fmt.Printf("f11 %d %s %d\n", a, b, c)
		return "", nil
	}('a', "a", 23)
	s := g(5, 8, 3)
	fmt.Printf("h %v\n", s)
	return a, "sf", 34.89, 8

}

func g(s ...int) []int {
	defer func(a rune, b string, c int) (string, error) {
		fmt.Printf("f11 %d %s %d\n", a, b, c)
		return "", nil
	}('a', "a", 23)
	defer func() int { fmt.Println("f10"); return 5 }()
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", []int{6, 3, 0}...)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", 6, 3, 0)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("b", 3)
	defer func(a string, b ...int) { fmt.Printf("f9 %q %v\n", a, b) }("a")
	defer func(a ...int) { fmt.Printf("f8 %v\n", a) }(9, 8, 7, 6, 5, 4, 3, 2, 1)
	defer func(a []int, b string, c float32) { fmt.Printf("f7 %v %s %f\n", a, b, c) }([]int{0}, "", 23.09471)
	defer func(a, b int) { fmt.Printf("f6 %d %d\n", a, b) }(9, 8)
	defer func(s []int) { fmt.Printf("f5 %v\n", s) }(nil)
	defer func(s string) { fmt.Printf("f4 %q\n", s) }("test123test")
	defer func(f float32) { fmt.Printf("f3 %f\n", f) }(9.640173)
	defer func(i int) { fmt.Printf("f2 %d\n", i) }(981632)
	defer func() { fmt.Println("f1") }()
	h()
	return s
}

func h() {
	defer func(a string, b int) {
		defer func() {
			fmt.Println("h.3")
			defer func(a float64, b string) {
				fmt.Printf("h.4 %f %q\n", a, b)
				defer func() {
					fmt.Println("h.6")
				}()
				defer func(a int) {
					fmt.Printf("h.5 %q\n", a)
				}(52)
			}(3.9, "b")
			fmt.Println("h.3")
		}()
		fmt.Printf("h.1 %q %d\n", a, b)
		defer func(a int) {
			fmt.Printf("h.2 %d\n", a)
		}(2)
	}("a", 56)
	fmt.Println("h")
}
