// errorcheck

package main

func main() {
	type Int int

	var _ []Int = []int{1, 2, 3} // ERROR `cannot use []int literal (type []int) as type []Int in assignment`

	var _ []int = []Int{3, 4, 5} // ERROR `cannot use []Int literal (type []Int) as type []int in assignment`

	type SliceInt1 []Int
	var s1 SliceInt1
	_ = s1

	type SliceInt2 []Int
	var s2 SliceInt2
	_ = s2

	s1 = s2 // ERROR `cannot use s2 (type SliceInt2) as type SliceInt1 in assignment`

}
