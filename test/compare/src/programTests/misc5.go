// run

package main

import "fmt"

var _ = f()

func f() int {
	fmt.Println("f called!")
	return 0
}

func main() {
}
