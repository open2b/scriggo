// paniccheck

package main

import (
	"io"
)

func main() {
	var x io.Reader
	_ = x.(io.Reader)
}
