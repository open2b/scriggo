// run

package main

func main() {
	_ = func(i int) {
		_ = func() { _ = i }
	}
}
