// run

package main

import "fmt"

var P = new(int)

func main() {
	*P++
	fmt.Println(*P)
}
