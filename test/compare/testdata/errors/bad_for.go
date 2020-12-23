// errorcheck

package main

func main() {
	for i := range []int{} { } // ERROR "i declared but not used"
}
