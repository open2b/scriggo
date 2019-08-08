// run

package main

import "fmt"

func f() []int {
	fmt.Printf("f")
	return []int{1, 2, 3}
}

func g() int {
	fmt.Printf("g")
	return 10
}

func main() {
	f()[1] = g()
}
