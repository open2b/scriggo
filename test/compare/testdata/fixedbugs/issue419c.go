// run

package main

import "fmt"

var i1 int = 42

var i2 *int = &i1

var i3 *int = &i1

func main() {
	fmt.Println(i2 == i3)
}
