// run

package main

import "fmt"

func f() {
	fmt.Print("f")
}

func main() {
	defer f()
}
