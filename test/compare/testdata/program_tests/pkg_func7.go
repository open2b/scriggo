// run

package main

import "fmt"

func f(a int) {
	return
}

func main() {
	a := 2
	f(3)
	b := 10
	c := a + b
	fmt.Print(c)
	return
}
