// skip : this test cannot be run with the given implementation of errorcheck https://github.com/open2b/scriggo/issues/247

// errorcheck

// Verify that the Go compiler will not
// die after running into an undefined
// type in the argument list for a
// function.
// Does not compile.

package main

func mine(int b) int { // ERROR "undefined.*b"
	return b + 2 // ERROR "undefined.*b"
}

func main() {
	mine()     // GCCGO_ERROR "not enough arguments"
	c = mine() // ERROR "undefined.*c|not enough arguments"
}
