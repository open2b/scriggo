// run

package main

import (
	"fmt"
)

func main() {
	a := "message"
	b := &a
	fmt.Print("*b: ", *b)
	c := *b
	fmt.Print(", c: ", c)
}
