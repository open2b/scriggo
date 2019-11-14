// errorcheck

package main

func F() (a, b, c int) {
	return 0, 0, 0
}

func main() {
	_, _ = F() // ERROR `assignment mismatch: 2 variables but F() returns 3 values`
}