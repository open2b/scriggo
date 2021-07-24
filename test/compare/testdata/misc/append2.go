// errorcheck

package main

type Int int
type IntSlice []int

func main() {
	var s []int
	_ = s
	_ = append(s, []bool{}...)      // ERROR `cannot use []bool{} (type []bool) as type []int in append`
	_ = append(s, map[int]int{}...) // ERROR `cannot use map[int]int{} (type map[int]int) as type []int in append`
	_ = append(s, 5 ...)    // ERROR `cannot use 5 (type untyped int) as type []int in append`
	_ = append(s, []Int{}...)       // ERROR `cannot use []Int{} (type []Int) as type []int in append`
	_ = append(s, IntSlice{}...)

	var t IntSlice
	_ = t
	_ = append(t, []bool{}...)      // ERROR `cannot use []bool{} (type []bool) as type []int in append`
	_ = append(t, map[int]int{}...) // ERROR `cannot use map[int]int{} (type map[int]int) as type []int in append`
	_ = append(t, 5 ...)   // ERROR `cannot use 5 (type untyped int) as type []int in append`
	_ = append(t, []Int{}...)      // ERROR `cannot use []Int{} (type []Int) as type []int in append`
	_ = append(t, []int{}...)

	var p []Int
	_ = p
	_ = append(p, []bool{}...)      // ERROR `cannot use []bool{} (type []bool) as type []Int in append`
	_ = append(p, map[int]int{}...) // ERROR `cannot use map[int]int{} (type map[int]int) as type []Int in append`
	_ = append(p, 5 ...)   // ERROR `cannot use 5 (type untyped int) as type []Int in append`
	_ = append(p, []int{}...)      // ERROR `cannot use []int{} (type []int) as type []Int in append`
	_ = append(p, []Int{}...)

	var q []interface{}
	_ = q
	_ = append(q, []bool{}...)
	_ = append(q, map[int]int{}...) // ERROR `cannot use map[int]int{} (type map[int]int) as type []interface {} in append`
	_ = append(q, 5 ...)   // ERROR `cannot use 5 (type untyped int) as type []interface {} in append`
	_ = append(q, []int{}...)
	_ = append(q, []Int{}...)
}
