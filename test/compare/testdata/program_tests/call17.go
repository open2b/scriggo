// run

package main

import "testpkg"

func main() {
	f := testpkg.ReturnFunction()
	println(f(5))
}
