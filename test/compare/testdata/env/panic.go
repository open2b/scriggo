// skip

// paniccheck

package main

import (
	"testpkg"
)

func main() {
	testpkg.Fatal("some error occurred")
}
