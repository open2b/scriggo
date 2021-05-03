// run

package main

import (
	"fmt"
	"testpkg"
)

var Center = "msg"

func main() {
	nativeC := testpkg.Center
	fmt.Println(nativeC)
	c := Center
	fmt.Println(c)
}
