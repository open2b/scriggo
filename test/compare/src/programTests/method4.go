// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	var b = *bytes.NewBufferString("content of (indirect) buffer")
	s := b.String()
	fmt.Print(s)
}
