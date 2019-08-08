// run

package main

import "fmt"

func f(a ...int) {
	fmt.Println("variadic:", a)
	return
}

func g(a, b string, c ...int) {
	fmt.Println("strings:", a, b)
	fmt.Println("variadic:", c)
	return
}

func h(a string, b ...string) int {
	fmt.Println("a:", a)
	return len(b)
}

func main() {
	f(1, 2, 3)
	g("x", "y", 3, 23, 11, 12)
	fmt.Println("h:", h("a", "b", "c", "d"))
}
