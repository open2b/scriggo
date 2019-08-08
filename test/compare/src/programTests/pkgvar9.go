// run

package main

import "fmt"

var A = F()

func F() int {
	fmt.Print("F has been called!")
	return 200
}

func main() {
	fmt.Print("main!")
}
