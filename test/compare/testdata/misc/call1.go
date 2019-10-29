// run

package main

import "fmt"

var A = 10

func F() {
	fmt.Println(A)
}

func main() {
	f := F
	f()
}
