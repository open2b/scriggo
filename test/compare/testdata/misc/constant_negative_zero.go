// run

package main

import "math"

func check(f float64) {
	if f != 0 || math.Signbit(f) {
		println("BUG: got", f, "want 0.0")
	}
}

func main() {
	const x = -0.0
	const y float64 = -0
	const z float32 = -0
	check(-0.0)
	check(x)
	check(y)
	check(float64(z))
	check(-math.SmallestNonzeroFloat64 / math.MaxFloat64)
	check(math.SmallestNonzeroFloat64 / -math.MaxFloat64)
	check(x + y)
	check(x - y)
	check(x * y)
	check(x / 1)
}
