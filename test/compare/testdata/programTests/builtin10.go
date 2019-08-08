// run

package main

import "fmt"

func main() {
	a := "hello"
	b := []int{1, 2, 3}
	c := []string{"", "x", "xy", "xyz"}
	l1 := len(a)
	l2 := len(b)
	l3 := len(c)
	fmt.Print(l1)
	fmt.Print(l2)
	fmt.Print(l3)
}
