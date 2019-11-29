// run

package main

func main() {
	a := 0
	func() {
		b := &a
		*b = 1
	}()
}
