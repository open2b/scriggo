// run

package main

import (
	"fmt"
)

func main() {
	var i interface{} = "hello"
	s := i.(string)
	s, ok := i.(string)
	fmt.Println(s, ok)
}
