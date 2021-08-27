// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

var Center = "msg"

func main() {
	nativeC := testpkg.Center
	fmt.Println(nativeC)
	c := Center
	fmt.Println(c)
}
