// run

package main

import (
	"fmt"
)

func f() (int, int) {
	fmt.Println("f called!")
	return 10, 20
}

func g() (a, b int) {
	return f()
}

func main() {
	g()
}
