// run

package main

func main() {
	var a *int = new(int)
	v := &*a
	_ = v
}
