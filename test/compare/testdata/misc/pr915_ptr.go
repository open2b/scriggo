// run

package main

import "fmt"

var P = new(int)

func main() {
	fmt.Println(*P)
	*P = 32
	fmt.Println(*P)
}
