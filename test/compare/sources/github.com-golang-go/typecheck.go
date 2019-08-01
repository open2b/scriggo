// errorcheck

// Verify that the Go compiler will not
// die after running into an undefined
// type in the argument list for a
// function.
// Does not compile.

package main

func mine(int b) int { return b + 2 } // ERROR "undefined.*b"


func main() {
	mine()     
	c = mine() 
}
