// errorcheck

package main

func main() {
	type Int int

	var _ []Int = []int{1, 2, 3} // ERROR `cannot use []int literal (type []int) as type []Int in assignment`

	var _ []int = []Int{3, 4, 5} // ERROR `cannot use []Int literal (type []Int) as type []int in assignment`
}
