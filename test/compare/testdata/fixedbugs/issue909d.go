// run

package main

import "fmt"

type T int

func main() {
	m := map[T]int{T(0): 42}
	fmt.Println(len(m))
	delete(m, T(0))
	fmt.Println(len(m))
}
