// run

package main

import (
	"fmt"
)

func main() {
	m := map[string]int{}
	fmt.Print(m, ",")
	m["one"] = 1
	m["four"] = 4
	fmt.Print(m, ",")
	m["one"] = 10
	fmt.Print(m)
}
