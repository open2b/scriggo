// run

// Test the defer statement.

// What is deferred?                             Package function
// Does the deferred function take arguments?    Yes, one int
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        Yes
// Are there more than one nested defer?         No

package main

import "fmt"

func f(a int) {
	fmt.Println("a is ", a)
}

func main() {
	defer f(2)
	defer f(10)
}
