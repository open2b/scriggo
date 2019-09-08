// run

package main

import "fmt"

func main() {
	a := 0
	f := func() int {
		a := 10
		b := 15
		return a + b
	}
	a = f()
	fmt.Print(a)
	return
}
