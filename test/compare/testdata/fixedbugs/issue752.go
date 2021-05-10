// errorcheck

package main

func main() {
	for i := range []int{0} {
		_ = i
		*i = 2 // ERROR `invalid indirect of i (type int)`
	}
}
