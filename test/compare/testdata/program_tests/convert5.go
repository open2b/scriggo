// run

package main

import (
	"fmt"
)

func main() {
	var f64 float64 = 123456789.98765321
	var f32 float32 = float32(f64)
	fmt.Print(f64, f32)
}
