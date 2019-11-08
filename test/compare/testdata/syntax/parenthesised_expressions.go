// compile

package main

func main() {
	_ = int(1)
	_ = int(1,)
	f(1)
	f(1,)
	g(1,2)
	g(1, []int{2}...)
	_ = make([]int, 2,)
	_ = len("",)
}

func f(i int) { }
func g(i int, ii ...int) { }
