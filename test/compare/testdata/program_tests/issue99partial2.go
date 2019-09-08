// run

package main

import "fmt"

var f = func() { fmt.Print("called") }

func main() {
	f()
}
