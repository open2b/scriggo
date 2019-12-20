// run

package main

import "fmt"

const a, b = iota + 3, iota + c
const c = a + 2

func main() {
	fmt.Println(a, b, c)
}
