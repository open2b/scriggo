// run

package main

import "fmt"

func main() {
	slice := []string{"abc", "def", "ghi"}
	fmt.Print(slice[0][1])
	fmt.Print(slice[0])
	fmt.Print(slice[1])
	out := slice[1][2]
	fmt.Print(out)
}
