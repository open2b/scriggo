// run

package main

import (
	"fmt"
)

func main() {
	var a int = 20
	fmt.Print(a << 2)
	fmt.Print(a >> 1)
	var b int8 = 12
	fmt.Print(b << 3)
	fmt.Print(b >> 1)
	fmt.Print((b << 5) >> 5)
}
