// run

package main

import "testpkg"

func main() {
	defer testpkg.SayHello()
}
