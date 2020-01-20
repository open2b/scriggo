// run

package main

import (
	"fmt"
	"math/big"
)

func main() {
	// x is -0
	x := 0.0
	x = -x

	// y is +0
	y := 0.0

	// z is -0
	z := 0.0
	z = -z

	// -y implemented as x - y is negative
	fmt.Println(big.NewFloat(x - y).Signbit())

	// -z implemented as x - z is positive
	fmt.Println(big.NewFloat(x - z).Signbit())
}
