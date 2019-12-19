// run

package main

import "fmt"

func main() {
	if true {
		fmt.Println("true: ok")
	}
	if false {
		panic("false")
	}
	if true && false {
		panic("true && false")
	}
}
