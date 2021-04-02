// run

package main

func f() (r string) {
	func() {
		r = ""
	}()
	return ""
}

func main() {
	f()
}
