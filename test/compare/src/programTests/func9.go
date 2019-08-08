// run

package main

func main() {
	a := 10
	f := func() {
		a := 10
		_ = a
	}
	b := 20
	_ = a
	_ = b
	_ = f
}
