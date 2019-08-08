// run

// Test the defer statement.

// What is deferred?                             Package function
// Does the deferred function take arguments?    No
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        No
// Are there more than one nested defer?         No

package main

import "fmt"

func f() {
	fmt.Print("f")
}

func main() {
	defer f()
}
