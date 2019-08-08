// run

// Test the defer statement.

// What is deferred?                             Function literal
// Does the deferred function take arguments?    Yes, one string
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        No
// Are there more than one nested defer?         No

package main

import "fmt"

func main() {
	defer func(s string) {
		fmt.Print(s)
	}("b")
	fmt.Print("a")
}
