// run

package main

import "fmt"

func typeof(i interface{}) string {
	return fmt.Sprintf("%T", i)
}

func testComplex() {
	{
		r := 10.0
		i := -3.0
		c := complex(r, i)
		fmt.Println(c, typeof(c))
	}
	{
		c := complex(432.3, 432.2)
		fmt.Println(c, typeof(c))
	}
	{
		c := complex(-2, 3)
		fmt.Println(c, typeof(c))
	}
	{
		c1 := complex(2.0, -432)
		c2 := complex(-3, 30)
		c := c1 + c2
		fmt.Println(c, typeof(c))
	}
	{
		fmt.Println((3 - 3i) + complex(10, 3))
		fmt.Println((3 - 3i) - complex(10, 3))
		fmt.Println(complex(3, 3))
		fmt.Println(complex(3, 3.033))
	}
	{
		r := float32(10.3)
		i := float32(-32)
		c := complex(r, i)
		fmt.Println(c, typeof(c))
	}
	{
		var c complex128 = complex(1, 2)
		fmt.Println(c, typeof(c))
	}
	{
		var c complex64 = complex(1, -3.02)
		fmt.Println(c, typeof(c))
	}
}

func testReal() {
	{
		c := complex(30, -45)
		r := real(c)
		fmt.Println(r, typeof(r))
	}
	{
		r := real(45 - 3i)
		fmt.Println(r, typeof(r))
	}
	{
		fmt.Println(real(complex(34.234, -4322)))
	}
	{
		var c complex64 = complex(-4, 5)
		r := real(c)
		fmt.Println(r, typeof(r))
	}
	{
		var r float32 = real(complex64(complex(20, 39)))
		fmt.Println(r, typeof(r))
	}
}

func testImag() {
	{
		c := complex(30, -45)
		i := imag(c)
		fmt.Println(i, typeof(i))
	}
	{
		i := imag(45 - 3i)
		fmt.Println(i, typeof(i))
	}
	{
		fmt.Println(imag(complex(34.234, -4322)))
	}
	{
		var c complex64 = complex(-4, 5)
		i := imag(c)
		fmt.Println(i, typeof(i))
	}
	{
		var i float32 = imag(complex64(complex(20, 39)))
		fmt.Println(i, typeof(i))
	}
}

func main() {
	testComplex()
	testReal()
	testImag()
}
