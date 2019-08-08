// run

package main

import "fmt"

func main() {
	var s = make([]func(), 1)
	s[0] = func() { fmt.Println("called") }
	s[0]()
}
