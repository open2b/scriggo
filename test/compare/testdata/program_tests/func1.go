// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	a := 5
	b, c := testpkg.Pair()
	fmt.Println(a, b, c)
	return
}
