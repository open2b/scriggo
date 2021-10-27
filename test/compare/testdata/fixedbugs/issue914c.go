// run

package main

import "fmt"

var x map[string]string

func main() {
	x = map[string]string{"a": "b"}
	y := x
	fmt.Println(x, y)
	x = nil
	fmt.Println(x, y)
}
