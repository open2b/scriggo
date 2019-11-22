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

	{
		var v uint = 32
		fmt.Println(1<<v == rune(2<<v))
		fmt.Println(1<<v != rune(2<<v))
		fmt.Println(1<<v > rune(2<<v))
	}

	{
		var v uint = 32
		fmt.Println(rune(2<<v) == 1<<v)
		fmt.Println(rune(2<<v) != 1<<v)
		fmt.Println(rune(2<<v) > 1<<v)
	}

	{
		var v uint = 32
		fmt.Println(1<<v == 2<<v)
		fmt.Println(1<<v != 2<<v)
		fmt.Println(1<<v > 2<<v)
	}

	{
		var v uint = 32

		fmt.Println((1 << v) == ('\002' << v))
		fmt.Println(('\002' << v) == (1 << v))

		fmt.Println((rune('\002' << v)) == (1 << v))
		fmt.Println((1 << v) == (rune('\002' << v)))
	}

}
