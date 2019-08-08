// run

package main

import "fmt"

func main() {
	defer func() {
		fmt.Print("c")
	}()
	defer func() {
		fmt.Print("b")
	}()
	fmt.Print("a")
}
