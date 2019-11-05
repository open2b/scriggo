// compile

package main

type Int int
type String string
type SliceInt []Int

func main() {
	var _ chan int
	var _ chan String
	var _ chan<- int
	var _ chan<- Int
}
