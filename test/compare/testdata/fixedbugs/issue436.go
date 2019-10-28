// run

package main

import (
	"fmt"
	"math"
)

func main() {
	var a uint64 = 4 // a small number

	// if b is interpreted as a signed integer then it's considered negative
	var b uint64 = math.MaxInt64 + 40

	fmt.Println(a < b) // must be true
	if a >= b {
		fmt.Println(">=") // wrong
	} else {
		fmt.Println("<") // correct
	}
}
