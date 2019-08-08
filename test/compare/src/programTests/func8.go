// run

package main

import "fmt"

func f() {
	panic("")
}

func main() {
	f := func() {
		fmt.Print("local function")
		// No panic.
	}
	f()
	return
}
