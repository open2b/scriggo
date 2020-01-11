// run

package main

import "math"

const (
	MaxFloat64             = 1.797693134862315708145274237317043567981e+308
	SmallestNonzeroFloat64 = 4.940656458412465441765687928682213723651e-324
)

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
	check(-SmallestNonzeroFloat64 / MaxFloat64)
	check(SmallestNonzeroFloat64 / -MaxFloat64)
	check(x + y)
	check(x - y)
	check(x * y)
	check(x / 1)
}
