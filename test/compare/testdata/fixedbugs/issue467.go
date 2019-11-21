// errorcheck

package main

type _ [...]int     // ERROR `use of [...] array outside of array literal`
var _ [...]int      // ERROR `use of [...] array outside of array literal`
func _([...]string) {} // ERROR `use of [...] array outside of array literal`

func main() {
	type _ [...]int          // ERROR `use of [...] array outside of array literal`
	var _ [...]int           // ERROR `use of [...] array outside of array literal`
	_ = func([...]string) {} // ERROR `use of [...] array outside of array literal`
}
