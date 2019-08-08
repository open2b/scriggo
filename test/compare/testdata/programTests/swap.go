// run

package main

import "fmt"

func main() {
	a := 10
	b := 20
	fmt.Println(a, b)
	a, b = a, b
	fmt.Println(a, b)
	a, b = b, a
	fmt.Println(a, b)
	b, a = a, b
	fmt.Println(a, b)
}
