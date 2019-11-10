// run

package main

import "fmt"

func main() {
	fmt.Print(f(0) == 255)
}

func f(u uint8) uint8 {
	u--
	return u
}
