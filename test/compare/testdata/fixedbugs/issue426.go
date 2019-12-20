// errorcheck

package main

const X = iota

func f(int) {}

func main() {
	f(iota) // ERROR `undefined: iota`
}
