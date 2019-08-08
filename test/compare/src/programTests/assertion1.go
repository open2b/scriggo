// run

package main

func main() {
	a := interface{}(10)
	n, ok := a.(int)

	_ = n
	_ = ok
	return
}
