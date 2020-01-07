// errorcheck

package main

type T struct{}

func main() {
	var t T
	_ = t
	_ = t._ // ERROR "^cannot refer to blank field or method$"
}
