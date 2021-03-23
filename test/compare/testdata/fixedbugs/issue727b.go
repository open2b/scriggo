// run

package main

import "fmt"

func main() {
	quadruplicate := func(i int) int {
		double := func() int { return i * 2 }
		return double() * 2
	}
	x := quadruplicate(3)
	fmt.Println(x)
}
