// run

package main

import "fmt"

func main() {
	var c complex64
	c = complex(32, 12)
	fmt.Println(real(c))
	fmt.Println(imag(c))
}
