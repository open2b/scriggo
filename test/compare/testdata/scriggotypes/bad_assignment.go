// skip

// errorcheck

package main

func main() {

	type Int int
	var i1 = Int(42) // defined     type
	var i2 = int(42) // predeclared type
	_ = i1
	_ = i2

	i1 = i2 // ERROR `cannot use i2 (type int) as type Int in assignment`

}
