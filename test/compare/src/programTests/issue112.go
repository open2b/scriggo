// run

package main

import "fmt"

func main() {
	fmt.Printf("%#v ", [3]int{1, 2, 3})
	a := [4]string{"a", "b", "c", "d"}
	fmt.Printf("%#v", a)
}
