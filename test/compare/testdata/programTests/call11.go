// run

package main

import (
	"fmt"
)

func f(a int, b int) {
	fmt.Println("a is", a, "and b is", b)
	return
}

func g() (int, int) {
	return 42, 33
}

func main() {
	f(g())
}
