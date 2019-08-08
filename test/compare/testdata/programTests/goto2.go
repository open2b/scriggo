// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("a")
	goto L
	fmt.Print("???")
L:
	fmt.Print("b")
}
