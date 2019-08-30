// run

package main

func main() {
	var s = make([]interface{}, 1)
	s[0] = func() {}
	f := s[0].(func())
	_ = f
}