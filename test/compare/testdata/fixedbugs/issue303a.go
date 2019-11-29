// paniccheck

package main

func main() {
	var a *int
	v := &*a
	_ = v
}
