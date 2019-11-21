// run

package main

import "fmt"

func main() {

	// Context is an empty interface type.
	{
		n := 1
		var i interface{} = 300 >> n
		fmt.Println(i)
	}

	// Context given by a 'var' declaration with type.
	{
		r := int(20)
		var s int32 = 1 << r
		fmt.Println(s)
	}

	// Context given by an explicit convertion.
	{
		r := int(20)
		s := int32(1 << r)
		fmt.Println(s)
	}

	// Context given by an unary operation.
	{
		r := int32(20)
		s := -(1 << r)
		fmt.Println(s)
	}

	// Context given by a function call.
	{
		f := func(a int32) {
			fmt.Println(a)
		}
		r := int64(10)
		f(1 << r)
	}

}
