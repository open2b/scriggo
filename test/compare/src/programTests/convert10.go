// run

package main

import (
	"fmt"
)

func main() {
	var a int = 400
	var b int8 = int8(a)
	var c int = int(b)
	fmt.Print(a, b, c)
}
