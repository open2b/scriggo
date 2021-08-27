// errorcheck

package main

import . "github.com/open2b/scriggo/test/compare/testpkg"

func main() {
	t := T(0)
	T.Method(&t) // ERROR `invalid method expression T.Method (needs pointer receiver: (*T).Method)`
	_ = t
}
