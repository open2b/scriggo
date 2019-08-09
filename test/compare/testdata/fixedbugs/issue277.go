// run

package main

import "fmt"

func f(args ...int) {
	fmt.Println(args)
	fmt.Println(len(args))
}

func g(args ...int) {
	fmt.Print(args)
}

func h(a, b int, args ...string) {
	fmt.Println(a, b, len(args), args)
}

func main() {
	f()
	g()
	f(1, 2, 3)
	g(1, 2, 3, 4, 5)
	h(1, 2)
	h(1, 2, "a", "b")
}
