// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	a := 5
	b, c := testpkg.Pair()
	fmt.Println(a, b, c)
	return
}
