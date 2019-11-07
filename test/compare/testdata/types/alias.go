// errorcheck

package main

func main() {
	type Int = int
	var _ Int = int(2)
	var _ int = Int(5)

	type SliceString = []string
	var _ SliceString = []string{}
	var _ SliceString = []int{} // ERROR `cannot use []int literal (type []int) as type []string in assignment`

	type Int2 = int
	var _ Int2 = Int(0)
	var _ Int2 = int(0)
	var _ Int2 = Int2(0)
	var _ Int2 = 0
	var _ Int2 = SliceString{} // ERROR `cannot use SliceString literal (type []string) as type int in assignment`
	var _ Int2 = string(30)    // ERROR `cannot use string(30) (type string) as type int in assignment`
	var _ Int2 = int32(0)      // ERROR `cannot use int32(0) (type int32) as type int in assignment`
}
