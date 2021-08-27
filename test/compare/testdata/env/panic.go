// paniccheck

package main

import (
	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {
	testpkg.Fatal("some error occurred")
}
