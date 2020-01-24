// paniccheck

package main

import (
	"io"
)

func main() {
	var x interface{}
	_ = x.(io.Reader)
}
