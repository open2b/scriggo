// errorcheck

package main

import "io"

func main() {

	v := 2
	_ = v
	_ = io.EOF

	var (
		_ float64   = (1 << v)                         // ERROR `invalid operation: 1 << v (shift of type float64)`
		_ int8      = (1000000 << v)                   // ERROR `constant 1000000 overflows int8`
		_           = (9999999999 << v) == rune(10<<v) // ERROR `constant 9999999999 overflows int32`
		_           = (1 << v) + (2 << v) + 2.1        // ERROR `invalid operation: 1 << v (shift of type float64)`
		_           = (1 << v) + 1i                    // ERROR `invalid operation: 1 << v (shift of type complex128)`
		_ int       = (1 << v) + 1i                    // ERROR `invalid operation: 1 << v (shift of type complex128)`
		_ int       = (1.0 << v) + int(5)
		_ int       = int(5) + (1.0 << v)
		_ io.Reader = 1 << v          // ERROR `cannot use 1 << v (type int) as type io.Reader in assignment`
		_ int8      = ('€' << v) << 5 // ERROR `constant 8364 overflows int8`
		_ int8      = 1 << (1.0 << v)
	)

}
