// skip : enable before merging with 'master'.

// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	b := bytes.NewBuffer([]byte{1, 2, 3, 4, 5})
	l := b.Len()
	fmt.Print(l)
}
