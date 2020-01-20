// run

package main

import (
	"fmt"
	"math"
)

func main() {
	x := float64(0.0)
	b := math.Float64bits(-x)
	fmt.Println(b)
}
