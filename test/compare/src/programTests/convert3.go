// run

package main

import (
	"fmt"
)

func main() {
	var s []byte = []byte{97, 98, 99}
	var c string = string(s)
	fmt.Print("s: ", s, ", c: ", c)
}
