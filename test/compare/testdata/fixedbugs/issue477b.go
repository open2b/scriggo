// errorcheck

package main

func main() {

	{
		var v uint
		_ = v
		_ = (1<<31)<<v + 'a'<<v + 1.2 // ERROR `invalid operation: (1 << 31) << v (shift of type float64)`
	}

	{
		var v uint
		_ = v
		var _ = (1<<31)<<v + 'a'<<v + 1.2 // ERROR `(1 << 31) << v (shift of type float64)`
	}

}
