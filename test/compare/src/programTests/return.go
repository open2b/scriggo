// run

package main

import (
	"fmt"
)

func f() interface{} {
	return "hey"
}

func main() {
	a := f()
	fmt.Println(a)
}
