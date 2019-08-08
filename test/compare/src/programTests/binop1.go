// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print(true && false)
	a := true
	b := true
	fmt.Print(a && b)
	var i interface{} = true
	fmt.Print(i)
}
