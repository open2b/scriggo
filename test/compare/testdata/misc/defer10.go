// run

// Test the defer statement.

// What is deferred?                             Function literal
// Does the deferred function take arguments?    Yes, one []int
// Is the deferred function variadic?            No
// Are there more than one sibling defer?        No
// Are there more than one nested defer?         No

// run

package main

import (
	"fmt"
)

func main() {
	defer func(s []int) {
		fmt.Println(s)
	}([]int{1, 2, 3})
}
