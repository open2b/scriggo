// run

package main

import "fmt"

const (
	a, b = iota + 3, iota + c
	c    = a + 2
)

func main() {
	fmt.Println(a, b, c)
}
