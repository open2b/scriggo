// run

package main

import "fmt"

func main() {

	u := uint8(128)
	i := int(u)
	fmt.Println(i) // should print 128, prints -128

}
