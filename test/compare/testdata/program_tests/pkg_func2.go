// run

package main

import (
	"fmt"
)

func floats() (float32, float64) {
	a := float32(5.6)
	b := float64(-432.12)
	return a, b
}

func main() {
	f1, f2 := floats()
	fmt.Println(f1, f2)
}
