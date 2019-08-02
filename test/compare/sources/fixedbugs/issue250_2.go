// run

package main

import "fmt"

var flag = "not set.."

func f() int {
	flag = "set!!!"
	return 20
}

var a = f()

func main() {
	fmt.Println("flag", flag)
	fmt.Println("a: ", a)
}
