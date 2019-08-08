// run

package main

import (
	"fmt"
)

func main() {
	a := 5
	b := 11
	xor := a ^ b
	bitclear := a &^ b
	fmt.Println(xor, bitclear)
}
