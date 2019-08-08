// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	var b = *bytes.NewBufferString("content of buffer")
	s := (&b).String()
	fmt.Print(s)
}
