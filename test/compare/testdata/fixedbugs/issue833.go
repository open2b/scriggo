// run

package main

func main() {
	type T struct {
		C int
	}
	var s1 struct {
		A int
		T
	}
	s1.A = 2
	println(s1.A)
	s1.C = 42
	println(s1.C)
}
