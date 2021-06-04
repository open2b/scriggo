// run

package main

import "fmt"

var i1 int

var i2 *int = &i1

func main() {
	*i2 = 32
	fmt.Println(i1, *i2)
}
