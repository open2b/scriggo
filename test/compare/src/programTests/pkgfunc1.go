// run

package main

import "fmt"

func inc(n int) int {
	return n + 1
}

func main() {
	var a, res, b, c int

	a = 2
	res = inc(8)
	b = 10
	c = a + b + res

	fmt.Print(c)
	return
}
