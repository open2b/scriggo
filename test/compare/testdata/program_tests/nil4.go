// run

package main

import (
	"fmt"
)

func getNil() interface{} {
	return nil
}

func main() {
	fmt.Println(getNil())
	n := getNil()
	fmt.Println(n)
}
