// errorcheck

package main

func main() {
	type String string
	t := []byte{}
	_ = t
	s := String("ciao")
	_ = s
	_ = 2 + append(t, s...) // ERROR `invalid operation: 2 + append(t, s...)`
}
