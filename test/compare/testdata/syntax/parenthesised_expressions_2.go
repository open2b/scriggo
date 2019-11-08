// errorcheck

package main

func main() {
	_ = int(1,,)    // ERROR `syntax error: unexpected comma, expecting expression`
	_ = g(1, ...) // ERROR `syntax error: unexpected ..., expecting expression`
	_ = 1,          // ERROR `syntax error: unexpected }, expecting expression`
}

func g(i int, ii ...int) { }
