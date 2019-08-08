// run

package main

import (
	"fmt"
)

func f(s []int) {
	fmt.Print(s)
}

func main() {
	f([]int{1, 2, 3})
	f(nil)
	f([]int{})
}
