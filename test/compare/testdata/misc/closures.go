// run

package main

func F() func() {
	a := 10
	return func() {
		b := &a
		*b = 30
		println(a)
	}
}

func main() {

	// 'f' contains a reference to 'a', which is declared in the parent block.
	{
		f := func() {}
		{
			a := 1
			f = func() { println(a) }
		}
		f()
	}

	// A deeply-nested version of the code above.
	{
		f := func() {}
		{
			{
				{
					a := 10
					b := 20
					var c int = 30
					f = func() { println(a, b, c) }
				}
			}
		}
		f()
	}

	// 'f' does not reference to the external 'a' cause it contains a new 'a'
	// definition inside his body.
	{
		f := func() {}
		{
			a := 1
			f = func() { a := 2; print(a) }
			_ = a
		}
		f()
	}

	// 'g' accesses to 'a' for writing.
	{
		a := 2
		g := func() { a = 10 }
		println(a)
		g()
		println(a)
	}

	// {
	// 	a := 2
	// 	g := func() {
	// 		b := &a
	// 		*b = 5
	// 	}
	// 	println(a)
	// 	g()
	// 	println(a)
	// }

	{
		F()()
	}
}
