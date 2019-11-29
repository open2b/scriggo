// errorcheck

package main

var b = int{1} // ERROR `invalid type for composite literal: int`

func main() {
	{
		var b = int{1} // ERROR `invalid type for composite literal: int`
	}
	{
		type T chan int
		_ = T{} // ERROR `invalid type for composite literal: T`
	}
	{
		_ = UndefType{} // ERROR `undefined: UndefType`
	}
}
