// run

package main

import "fmt"

func main() {
	f := func(i int) int {
		x := func() int { return i }
		return x()
	}
	fmt.Println(f(7))
}
