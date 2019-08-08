// run

package main

import (
	"fmt"
)

func main() {
	a := [3]int{1, 2, 3}
	b := [...]string{"a", "b", "c", "d"}
	fmt.Println(a, b)
	fmt.Print(len(a), len(b))
}
