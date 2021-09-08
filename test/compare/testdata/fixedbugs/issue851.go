// errorcheck

package main

func main() {
	var x = { "one": 1, "two": 2, "three": 3 } // ERROR `syntax error: unexpected {, expecting expression`
}
