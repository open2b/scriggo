// skip

// run

package main

import (
	"bytes"
	"fmt"
)

func main() {
	b := bytes.NewBufferString("message")
	l := (*bytes.Buffer).Len(b)
	fmt.Print("length of ", b, " is ", l)
}
