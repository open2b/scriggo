// run

package main

import "fmt"

func f() int {
	fmt.Print("f")
	return 0
}

func main() {
	_ = 4
	_ = f()
	_ = []int{f(), f(), 4, 5}
	_ = f() + f()
	_ = -f()
}
