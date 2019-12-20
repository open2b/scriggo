// run

package main

import (
	"fmt"
	"testpkg"
)

func main() {
	t := true
	f := false
	if t {
		fmt.Println("true: ok")
	}
	if f {
		panic("false")
	}
	if t && f {
		panic("true && false")
	}
	if !t {
		panic("!t")
	}
	if !f {
		fmt.Println("!f: ok")
	}
	if testpkg.BooleanValue {
		fmt.Println("testpkg.BooleanValue is true")
	} else {
		fmt.Println("testpkg.BooleanValue is false")
	}
}
