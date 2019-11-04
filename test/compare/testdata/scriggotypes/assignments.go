// errorcheck

package main

func main() {

	type Int1 int
	type Int2 int

	var i1 = Int1(42)
	var i2 = Int2(42)

	_, _ = i1, i2

	i1 = i1 // Int1, Int1
	i2 = i2 // Int2, Int2
	i1 = 22 // Int1, untyped constant

	i1 = i2 // ERROR `cannot use i2 (type Int2) as type Int1 in assignment`
	i2 = i1 // ERROR `cannot use i1 (type Int1) as type Int2 in assignment`

	i1 = i1 // same scriggo type, ok
	i2 = i2 // same scriggo type, ok

	var i int = i1 // ERROR `cannot use i1 (type Int1) as type int in assignment`

}
