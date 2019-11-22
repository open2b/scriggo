// errorcheck

package main

func main() {

	v := 2
	_ = v

	var _ float64 = (1 << v)    // ERROR `invalid operation: 1 << v (shift of type float64)`
	var _ int8 = (1000000 << v) // ERROR `constant 1000000 overflows int8`

	// TODO: should return error.
	//var _ = (9999999999 << v) == rune(10<<v) // E RROR `constant 9999999999 overflows rune`

	// TODO: should be                         `invalid operation: 1 << v (shift of type float64)`
	var _ = (1 << v) + (2 << v) + 2.1 // ERROR `invalid operation: (1 << v + 2 << v) + 2.1 (constant 2.1 truncated to integer)`

	// TODO: should be             `invalid operation: 1 << v (shift of type complex128)`
	var _ = (1 << v) + 1i // ERROR `invalid operation: 1 << v + 1i (failed type conversion)`

	// TODO: should be                 `constant 1i truncated to integer`
	var _ int = (1 << v) + 1i // ERROR `invalid operation: 1 << v + 1i (failed type conversion)`

}
