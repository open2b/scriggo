// skip

// paniccheck

package main

func main() {
	var i interface{} = 10
	_ = i.(string)
}
