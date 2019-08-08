// run

package main

import (
	"fmt"
)

func f(a int, b int) {
	fmt.Println("a is", a, "and b is", b)
}

func g(s string) (int, int) {
	return len(s), len(s) * (len(s) + 7)
}

func main() {
	f(g("a string!"))
}
