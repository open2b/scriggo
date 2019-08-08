// run

package main

import "fmt"

func main() {
	fmt.Println("a")
	defer func() {
		fmt.Println("c")
		defer func() {
			fmt.Println("e")
		}()
		fmt.Println("d")
	}()
	fmt.Println("b")
}
