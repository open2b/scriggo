// run

package main

func main() {
	type S struct{}
	_ = func() {
		_ = S{}
	}
}
