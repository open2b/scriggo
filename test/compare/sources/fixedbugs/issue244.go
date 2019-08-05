// run

package main

import "fmt"

func main() {

	// https://github.com/open2b/scriggo/issues/258
	// var _ = nil == interface{}(nil)

	_ = nil == interface{}(nil)
	_ = interface{}(nil) == nil
	a := interface{}(nil) == nil
	_ = a
	fmt.Print(interface{}(nil) == nil)
}
