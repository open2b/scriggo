// run

// Test the defer statement.

// What is deferred?                             Function literal
// Does the deferred function take arguments?    No
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        Yes, two
// Are there more than one nested defer?         No

package main

import "fmt"

func main() {
	defer func() {
		fmt.Print("c")
	}()
	defer func() {
		fmt.Print("b")
	}()
	fmt.Print("a")
}
