// run

package main

import "github.com/open2b/scriggo/test/compare/testpkg"

func main() {
	f := testpkg.ReturnFunction()
	println(f(5))
}
