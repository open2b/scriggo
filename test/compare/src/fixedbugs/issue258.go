// compile

package main

var _ = "g1"
var _, _ = "g2", "g3"

func main() {
	var _ = nil == interface{}(nil)
	var _, _ = 1, 2
	_ = 3
	_, _, _ = 4,5,6
}