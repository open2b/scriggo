// errorcheck

package main

type t struct {
	x int
	x int // ERROR `duplicate field x`
}

func main() {}
