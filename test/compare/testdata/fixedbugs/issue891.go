// errorcheck

package main

type Float64 float64

func main() {
	var f64 float64
	var F64 Float64
	_ = complex(F64, f64) // ERROR `invalid operation: complex(F64, f64) (mismatched types Float64 and float64)`
	_, _ = f64, F64
}