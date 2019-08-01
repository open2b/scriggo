// run

package main

import "fmt"

func init() {
	fmt.Println("init 1")
}

func init() {
	fmt.Println("init 2")
}

func main() {
}

func init() {
	fmt.Println("init 3")
}
