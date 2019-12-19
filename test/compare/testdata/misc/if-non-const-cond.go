// run

package main

import "fmt"

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
}
