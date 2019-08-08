// run

package main

import (
	"fmt"
	"testpkg"
)

var Center = "msg"

func main() {
	predefinedC := testpkg.Center
	fmt.Println(predefinedC)
	c := Center
	fmt.Println(c)
}
