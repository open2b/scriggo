// run

package main

import "fmt"

func pair() (int, float64) {
	return 42, 33.0
}

func main() {
	a := 2
	b, c := pair()
	d, e := 11, 12
	fmt.Print(a, b, c, d, e)
	return
}
