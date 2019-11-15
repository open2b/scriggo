// run

package main

type Int int
type String string
type SliceInt []Int

func main() {
	var _ chan int
	var _ chan String
	var _ chan<- int
	var _ chan<- Int
	var _ chan<- SliceInt
	
	c1 := make(chan Int)
	_ = c1

	c2 := make(chan SliceInt)
	_ = c2

	c3 := make([](chan [3]String), 10)
	_ = c3
}
