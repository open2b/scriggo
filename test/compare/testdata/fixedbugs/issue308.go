// errorcheck

package main

func main() {
	i := 0
	c := ""
	for i, c = range "ab" { } // ERROR `cannot assign int32 to c (type string) in multiple assignment`
	_ = i
	_ = c
}
