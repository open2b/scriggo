// paniccheck

package main

func main() {
	var i interface{}
	_ = i.(interface{})
}
