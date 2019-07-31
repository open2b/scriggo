// runcompare

package main

import (
	"fmt"
	"math"
)

func main() {
	var x, y int = 3, 4
	var f float64 = math.Sqrt(float64(x*x + y*y))
	var z uint = uint(f)
	fmt.Println(x, y, z)

	m := map[string]interface{}{}
	m1 := map[string]interface{}(m)
	_ = m1
}
