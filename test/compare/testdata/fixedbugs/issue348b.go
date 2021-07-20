// run

package main

var s = "package"

func main() {
	s := "local"
	func() {
		_ = s
	}()
}
