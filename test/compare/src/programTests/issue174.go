// run

package main

import "fmt"

func f(a string, slice ...int) {
	fmt.Print(slice)
	return
}

func main() {
	f("a", 10, 20, 30)
}
