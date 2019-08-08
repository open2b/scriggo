// run

package main

func main() {
	a, b := 1, 1
	_ = a + b
	_ = func() {
		_ = a + b
	}
}
