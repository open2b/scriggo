// errorcheck

package main

func main() {

	v := 2
	_ = v

	var _ float64 = (1 << v)          // ERROR `invalid operation: 1 << v (shift of type float64)`
	// var _ = (1 << v) + (2 << v) + 2.1 // E RROR `invalid operation: 1 << v (shift of type float64)`
	var _ int8 = (1000000 << v)       // ERROR `constant 1000000 overflows int8`

}
