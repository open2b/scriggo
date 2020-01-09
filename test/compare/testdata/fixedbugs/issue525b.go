// run

package main

import "fmt"

func main() {
	s := []byte("000000")
	copy(s, "12346")
	fmt.Println(s)
}
