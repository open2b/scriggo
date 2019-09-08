// run

package main

import (
	"fmt"
)

func main() {
	var i interface{} = interface{}(nil)
	fmt.Printf("i: %v, type(i): %T", i, i)
}
