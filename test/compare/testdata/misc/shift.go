// run

package main

import "fmt"

func main() {

	{
		n := 1
		var i interface{} = 300 >> n
		fmt.Println(i)
	}

	{
		r := int(20)
		var s int32 = 1 << r
		fmt.Println(s)
	}

	{
		r := int(20)
		s := int32(1 << r)
		fmt.Println(s)
	}

	{
		r := int32(20)
		s := -(1 << r)
		fmt.Println(s)
	}

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

	{
		var v uint = 32

		fmt.Println((1 << v) + ('\002' << v))
		fmt.Println(('\002' << v) + (1 << v))

		fmt.Println((rune('\002' << v)) + (1 << v))
		fmt.Println((1 << v) + (rune('\002' << v)))
	}

	{
		v := 32
		fmt.Println(
			(1 << v) + int(10 << (v*2)) + 
			(200 << v) + int('a' << (v*2)),
		)
	}

}
