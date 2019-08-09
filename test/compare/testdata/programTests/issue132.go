//+build go1.13

// run

package main

import "fmt"

var (
	_ = d
	_ = f("_", c, b)
	a = f("a", 0, 0)
	b = f("b", 0, 0)
	c = f("c", 0, 0)
	d = f("d", 0, 0)
)

func f(s string, a int, b int) int {
	fmt.Print(s, a, b)
	return 0
}

func main() {}
