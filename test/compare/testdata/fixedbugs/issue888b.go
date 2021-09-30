// run

package main

import "fmt"

func main() {
	var c complex128
	c = complex(32, 12)
	r := real(c)
	i := imag(c)
	fmt.Println(r, i)
}
