// run

package main

import (
	"fmt"
)

func main() {
	src := []int{1, 2, 3}
	dst := []int{0, 0, 0}
	n := copy(dst, src)
	fmt.Print("n is ", n)
	fmt.Print(", n is now ", copy(dst, src))
}
