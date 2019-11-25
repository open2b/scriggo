// run

package main

import "fmt"

func f() (a, b int) {
	a, b = 1, 2
	return b, a
}

func main() {
	fmt.Println(f())
}