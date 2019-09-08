// run

package main

import (
	"fmt"
)

func main() {
	m := map[string]int{"one": 1, "three": 0}
	v1, ok1 := m["one"]
	v2, ok2 := m["two"]
	v3, ok3 := m["three"]
	fmt.Print(v1, v2, v3, ",")
	fmt.Print(ok1, ok2, ok3)
}
