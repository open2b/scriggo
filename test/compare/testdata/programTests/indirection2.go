// run

package main

import (
	"fmt"
)

func main() {
	a := "a"
	b := &a
	c := *b
	fmt.Print(c)
}
