// run

package main

import "fmt"

type Int = int

func zero() Int {
	var z int
	return z
}

func main() {
	fmt.Print(zero())
	var z1 Int = zero()
	var z2 int = zero()
	fmt.Print(z1, z2)
}
