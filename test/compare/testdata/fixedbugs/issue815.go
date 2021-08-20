// errorcheck

package main

func main() {
	var _ = 1e100000000 // ERROR `constant 1e+100000000 overflows float6`
}