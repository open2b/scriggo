// errorcheck

package main

func main() {
	var a int
	a = x default 5 // ERROR "unexpected default at end of statement"
	_ = a
}
