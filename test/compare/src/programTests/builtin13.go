// run

package main

import (
	"fmt"
)

func main() {
	m := map[string]int{}
	fmt.Print(len(m))
	m["one"] = 1
	fmt.Print(len(m))
	m["one"] = 1
	m["one"] = 2
	fmt.Print(len(m))
	m["one"] = 1
	m["two"] = 2
	m["three"] = 3
	m["five"] = 4
	fmt.Print(len(m))
}
