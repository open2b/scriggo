// run

package main

func f() (r int) {
	func() {
		r = 1
	}()
	return
}

func main() {
	_ = f()
}
