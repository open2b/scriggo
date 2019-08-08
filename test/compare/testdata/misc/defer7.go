// run

package main

import "fmt"

func f(a int) {
	fmt.Println("a is ", a)
}

func main() {
	defer f(2)
	defer f(10)
}
