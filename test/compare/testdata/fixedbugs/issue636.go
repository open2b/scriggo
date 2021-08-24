// errorcheck

package main

type T1 struct {
	MyField int
}

type T2 struct {
	MyField int
}

type T struct {
	T1
	T2
}

func main() {
	var b T
	_ = b.MyField // ERROR `ambiguous selector b.MyField`
	_ = b
}
