// errorcheck

package main

func _() {}
func _() {}

var _ int

const _ = 0

func _() {}

func main() {
	_() // ERROR `cannot use _ as value`
}
