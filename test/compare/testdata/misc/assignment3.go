// errorcheck

package main

func main() {

	{
		a := 10
		_ = a
		a, b := "string", 20 ; _ = b // ERROR `cannot use "string" (type string) as type int in assignment`		
	}

	{
		var a int = 1 << 63; _ = a // ERROR `constant 9223372036854775808 overflows int`
		var a = 1 << 63; _ = a     // ERROR `constant 9223372036854775808 overflows int`
		a := 1 << 63 ; _ = a       // ERROR `constant 9223372036854775808 overflows int`
		_ = 1 << 63                // ERROR `constant 9223372036854775808 overflows int`
		const _ int = 1 << 100     // ERROR `constant 1267650600228229401496703205376 overflows int`
	}

	{
		a := 10
		a = nil // ERROR `cannot use nil as type int in assignment`
		_ = a
	}
	{
		_ = nil // ERROR `use of untyped nil`
	}
	{
		var a int32 = 10
		_ = a
	}
	{
		a, b := 10, 20
		_, _ = a, b
		a, c := nil, 30 ; _ = c // ERROR `cannot use nil as type int in assignment`
	}
	{
		var a int8
		a >>= uint8(1)
	}
	{
		var a int8
		a <<= uint8(1)
	}
	{
		var s string
		// TODO: fix the error message:
		s >>= 10 // ERROR `invalid operation: s10 (shift of type string)`
		_ = s
	}

}
