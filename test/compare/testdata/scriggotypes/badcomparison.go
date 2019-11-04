// skip

// errorcheck

package main

func main() {

	type Int int
	var i1 = Int(42) // defined     type
	var i2 = int(42) // predeclared type
	_ = i1
	_ = i2

	_ = i1 == i2 // ERROR `invalid operation: i1 == i2 (mismatched types Int and int)`

}
