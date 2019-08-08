// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	b := *bytes.NewBuffer([]byte{97, 98, 99})
	fmt.Printf("b has type %T", b)
	l := b.Len()
	fmt.Print(", l is ", l)
}
