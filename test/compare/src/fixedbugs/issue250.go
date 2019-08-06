// run

package main

import "fmt"

var flag = "not set"

func f() int {
	flag = "set!!!"
	return 0
}

var a = f()

func main() {
	fmt.Print(flag)
}
