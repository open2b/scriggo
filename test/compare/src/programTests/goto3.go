// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print("a")
	goto L
L:
	fmt.Print("b")
}
