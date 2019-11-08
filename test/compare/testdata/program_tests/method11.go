// skip : enable before merging with 'master'.

// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	b := bytes.NewBuffer([]byte{97, 98, 99})
	lenMethod := b.Len
	l := lenMethod()
	fmt.Print("l is ", l)
}
