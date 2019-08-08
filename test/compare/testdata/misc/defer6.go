// run

// Test the defer statement.

// What is deferred?                             Function literal
// Does the deferred function take arguments?    No
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        No
// Are there more than one nested defer?         Yes, one level of depth

package main

import "fmt"

func main() {
	fmt.Println("a")
	defer func() {
		fmt.Println("c")
		defer func() {
			fmt.Println("e")
		}()
		fmt.Println("d")
	}()
	fmt.Println("b")
}
