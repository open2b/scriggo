// runcmp

package main

import (
	"fmt"
)

/* This is a package-level comment
 */

// This too.

func main() {
	i := 0

	fmt.Println(i)
	i++

	// a comment

	fmt.Println(i)
	i++

	/* a block comment */

	fmt.Println(i)
	i++

	/*
	 * a multi-line
	 * comment
	 */

	fmt.Println(i)
	i++

	/*

		fmt.Println("?")
	*/

	fmt.Println(i)
	i++

	/*
		// a comment inside comment
	*/

	a := 2 // declaration and initialization
	fmt.Println(a)

	fmt.Println(a /* this is ignored */ + 3) // this prints 5

	a = /*                        */ 10
	fmt.Println(a) /******/

}
