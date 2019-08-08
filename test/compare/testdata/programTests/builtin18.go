// run

package main

import (
	"fmt"
)

func main() {
	pi := new(string)
	pv := *pi
	fmt.Print("*pi:", pv, ",")
	*pi = "newv"
	pv = *pi
	fmt.Print("*pi:", pv)
}
