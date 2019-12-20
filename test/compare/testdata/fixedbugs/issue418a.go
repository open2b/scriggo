// run

package main

import "fmt"

func main() {
	b := 0
	f := func() int {
		b--
		return -b
	}
	x, y := f(), f()
	fmt.Println(x, y, b)
}
