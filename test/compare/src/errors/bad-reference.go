// errorcheck

package main

func main() {
	_ = &0 // ERROR `cannot take the address of 0`
}
