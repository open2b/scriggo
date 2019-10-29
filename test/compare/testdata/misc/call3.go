// run

package main

import "fmt"

var A = 10

func F() {}

func main() {
	fmt.Println(A)
	f := F
	f()
	fmt.Println(A)
}
