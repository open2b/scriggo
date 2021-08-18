// errorcheck

package main

func main() {}

func f() {
	defer complex(1, 2) // ERROR `defer discards result of complex(1, 2)`
}
