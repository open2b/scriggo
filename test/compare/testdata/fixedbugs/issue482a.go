// run

package main

func main() {

	var i int
	//var f float64
	//var c complex128
	//var s string
	var b bool

	// constant + constant.
	_ = 1 + 2
	_ = 1 + int(2)
	_ = int(1) + 2
	_ = int(1) + int(2)

	// untyped + untyped.
	_ = 1 + (1 << i)
	_ = 1 + ('a' << i)
	_ = (1 << i) + 1
	_ = ('a' << i) + 1

	// typed + untyped.
	_ = i + 2
	_ = i + (1 << i)
	_ = i + ('a' << i)
	_ = i + (1.0 << i)
	_ = i + (0i << i)
	_ = int(1) + (1 << i)
	_ = int(1) + ('a' << i)
	_ = int(1) + (1.0 << i)
	_ = int(1) + (0i << i)

	// untyped + typed.
	_ = 2 + i
	_ = (1 << i) + i
	_ = ('a' << i) + i
	_ = (1.0 << i) + i
	_ = (0i << i) + i
	_ = (1 << i) + int(1)
	_ = ('a' << i) + int(1)
	_ = (1.0 << i) + int(1)
	_ = (0i << i) + int(1)

	// typed + typed.
	_ = i + i
	_ = int(1) + i
	_ = i + int(2)

	// shifts.
	_ = 1 << 2
	_ = 1 << int(2)
	_ = 1 << (1 << i)
	_ = 1 << i
	_ = int(1) << 2
	_ = int(1) << int(2)
	_ = int(1) << (1 << i)
	_ = int(1) << i
	_ = (1 << i) << 2
	_ = (1 << i) << int(2)
	_ = (1 << i) << (1 << i)
	_ = (1 << i) << i
	_ = i << 2
	_ = i << int(2)
	_ = i << (1 << i)
	_ = i << i

	// booleans.
	_ = true == false
	_ = true == bool(false)
	_ = true == (1 == 2)
	_ = true == b
	_ = bool(true) == false
	_ = bool(true) == bool(false)
	_ = bool(true) == (1 == 2)
	_ = bool(true) == b
	_ = (1 == 2) == false
	_ = (1 == 2) == bool(false)
	_ = (1 == 2) == (1 == 2)
	_ = (1 == 2) == b
	_ = b == false
	_ = b == bool(false)
	_ = b == (1 == 2)
	_ = b == b

	// less on booleans.

	// nil.

	// interfaces.
	_ = 1 == interface{}(1)
	_ = int(1) == interface{}(1)
	_ = (1 << i) == interface{}(1)
	_ = i == interface{}(1)
	_ = interface{}(1) == 1
	_ = interface{}(1) == int(1)
	_ = interface{}(1) == (1 << i)
	_ = interface{}(1) == i

}
