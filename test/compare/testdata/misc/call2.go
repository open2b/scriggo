// run

package main

import "fmt"

var A = 20

func main() {
	(func() {})()
	fmt.Println(A)
}
