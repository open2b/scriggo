// errorcheck

package main

func main() {
	var x interface{}
	_ = x
	_ = x.() // ERROR `syntax error: unexpected ), expecting type`
}
