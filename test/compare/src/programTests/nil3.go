// run

package main

import (
	"fmt"
)

func f(s []int) {
	fmt.Printf("s: %v, len(s): %d, type(s): %T\n", s, len(s), s)
}

func main() {
	f([]int{10, 20, 30})
	f(nil)
}
