// run

package main

import "fmt"

func f() {
	fmt.Println("f")
}

func g() {
	fmt.Println("g")
}

func main() {
	defer f()
	defer g()
}
