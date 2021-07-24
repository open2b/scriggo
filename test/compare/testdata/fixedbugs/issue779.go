// errorcheck

package main

func main() {

	type E1 struct {
		x int
	}
	type E2 struct {
		x int
	}
	type S struct {
		x E1
	}
	type T struct {
		x E2
	}

	var s S
	_ = s
	var t T
	_ = t

	s = S(t) // ERROR `cannot convert t (type T) to type S`

}
