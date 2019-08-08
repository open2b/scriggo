// run

package main

import (
	"fmt"
)

func f(s string) int {
	fmt.Print(s, " ", "side-effect, ")
	return 10
}

func main() {
	_ = []int{f("slice"), f("slice"), 5, f("slice")}
	_ = [...]int{10: f("array")}
	_ = map[string]int{"f": f("map")}
}
