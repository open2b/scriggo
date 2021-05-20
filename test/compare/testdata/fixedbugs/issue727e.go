// run

package main

import "fmt"

func main() {
	f := func(i int) int {
		ref := &i
		*ref = 10 * i
		return i
	}
	fmt.Println(f(77))
}
