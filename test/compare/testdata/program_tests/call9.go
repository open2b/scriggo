// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	a := 2
	a = testpkg.G11(9)
	fmt.Print(a)
	return
}
