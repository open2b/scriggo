// errorcheck

package main

func f() {}

func main() {
	(	// ERROR `syntax error: unexpected newline, expecting )`
		f()
}
