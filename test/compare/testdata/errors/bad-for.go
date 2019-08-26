// errorcheck

package main

func main() {
	for i := range []int{} { } // ERROR "i declared and not used"
}
