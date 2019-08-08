// run

package main

import (
	"fmt"
)

func main() {
	var str = string("hello!")
	var slice = []byte(str)
	fmt.Print("str: ", str, ", slice: ", slice)
}
