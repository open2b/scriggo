// run

package main

import "fmt"

func main() {
	test1()
}

func test1() {
	s := []byte("ciao")
	fmt.Println(s)
}

func test2() {
	s := "a string"
	sb := []byte(s)
	fmt.Println(sb)
	fmt.Println(string(sb))
}
