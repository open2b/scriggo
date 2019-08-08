// run

package main

import (
	"fmt"
)

func main() {
	fmt.Print(len([]int{1, 2, 3}))
	fmt.Print(cap(make([]int, 1, 10)))
}
