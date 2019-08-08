// run

package main

import (
	"fmt"
)

var append = func() int { return 10 }

var cap = []int{}

func close() {}

const bool = "bool"

func main() {
	fmt.Println(append())
	fmt.Println(cap)
	close()
	fmt.Println(bool)
}
