// run

// Note that code worked as expected even before fixing the issue #594.

package main

import "fmt"

func g(e interface{}) (int, int) {
	return 10, 20
}

func f(d interface{}) (int, int) {
	return g(d)
}

func main() {
	fmt.Println(f(50))
}
