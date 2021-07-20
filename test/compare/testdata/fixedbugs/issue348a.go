// run

package main

import "fmt"

var s = struct{ A int }{20}

func main() {
	s := struct{ A int }{40}
	func() {
		_ = s
		s.A = 20
	}()
	fmt.Println(s)
}
