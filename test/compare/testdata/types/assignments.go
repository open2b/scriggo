// errorcheck

package main

func arrayTypes() {
	type Int1 int
	var a1 [3]Int1
	_ = a1
	a1[0] = int(0) // ERROR `cannot use int(0) (type int) as type Int1 in assignment`
	a1 = [3]int{}  // ERROR `cannot use [3]int{} (type [3]int) as type [3]Int1 in assignment`
}

func chanTypes() {

	return // avoids operations on channels

	type Int int
	type String string

	var c1 chan int
	_ = c1
	c1 <- Int(0)           // ERROR `cannot use Int(0) (type Int) as type int in send`
	var _ Int = <-c1       // ERROR `cannot use <-c1 (type int) as type Int in assignment`
	c1 = (chan Int)(nil)   // ERROR `cannot use (chan Int)(nil) (type chan Int) as type chan int in assignment`
	c1 = (chan<- Int)(nil) // ERROR `cannot use (chan<- Int)(nil) (type chan<- Int) as type chan int in assignment`
	c1 = (<-chan Int)(nil) // ERROR `cannot use (<-chan Int)(nil) (type <-chan Int) as type chan int in assignment`

	var c2 chan Int
	_ = c2
	c2 <- Int(0)
	c2 <- int(42)    // ERROR `cannot use int(42) (type int) as type Int in send`
	var _ int = <-c2 // ERROR `cannot use <-c2 (type Int) as type int in assignment`

	var c3 chan<- []String
	_ = c3
	c3 <- []Int{} // ERROR `cannot use []Int{} (type []Int) as type []String in send`

	var c4 (<-chan map[String][]Int)
	_ = c4
	var _ int = <-c4 // ERROR `cannot use <-c4 (type map[String][]Int) as type int in assignment`

}

func definedTypes() {
	type Int1 int
	type Int2 int

	var i1 = Int1(42)
	var i2 = Int2(42)

	_, _ = i1, i2

	i1 = i1 // Int1, Int1
	i2 = i2 // Int2, Int2
	i1 = 22 // Int1, untyped constant

	i1 = i2 // ERROR `cannot use i2 (type Int2) as type Int1 in assignment`
	i2 = i1 // ERROR `cannot use i1 (type Int1) as type Int2 in assignment`

	i1 = i1 // same scriggo type, ok
	i2 = i2 // same scriggo type, ok

	var i int = i1 // ERROR `cannot use i1 (type Int1) as type int in assignment`

	i1 = int(42)         // ERROR `cannot use int(42) (type int) as type Int1 in assignment`
	i2 = float64(56.034) // ERROR `cannot use float64(56.034) (type float64) as type Int2 in assignment`

	type SliceByte []byte
	var _ SliceByte = SliceByte([]byte{1,2,3})
}

func mapTypes() {

	type Int int
	type String string
	type MapIntString map[int]string

	var m MapIntString
	_ = m
	m = map[int]int{}    // ERROR `cannot use map[int]int{} (type map[int]int) as type MapIntString in assignment`
	m = map[Int]String{} // ERROR `cannot use map[Int]String{} (type map[Int]String) as type MapIntString in assignment`
	m = []MapIntString{} // ERROR `cannot use []MapIntString{} (type []MapIntString) as type MapIntString in assignment`
}

func sliceTypes() {
	type Int int
	type String string
	var s []Int
	_ = s
	s = []int{}    // ERROR `cannot use []int{} (type []int) as type []Int in assignment`
	s = []String{} // ERROR `cannot use []String{} (type []String) as type []Int in assignment`
}

func ptrTypes() {
	type Int int
	type Ptrint *int
	type PtrInt *Int
	var p1 Ptrint
	_ = p1
	p1 = (*int)(nil)
	p1 = (Ptrint)(nil)
	p1 = (PtrInt)(nil) // ERROR `cannot use PtrInt(nil) (type PtrInt) as type Ptrint in assignment`
	p1 = (*Int)(nil)   // ERROR `cannot use (*Int)(nil) (type *Int) as type Ptrint in assignment`
}

func structTypes() {
	type Int int
	type String string
	type Field = map[Int][]string

	var s struct{ A, B Field }
	_ = s
	//s.A = 4              // ERROR `cannot use 4 s(type int) as type map[Int][]string in assignment` TODO
	//s.A = String("ciao") // ERROR `cannot use String("ciao") (type String) as type map[Int][]string in assignment`
	//s = []int{}          // ERROR `cannot use []int{} (type []int) as type struct { A map[Int][]string; B map[Int][]string } in assignment`

	type S struct{ A map[string]Field }
	var s2 S
	_ = s2
	//s2.A = 4                // ERROR `cannot use 4 (type int) as type map[string]map[Int][]string in assignment`
	//s2.A = map[string]Int{} // ERROR `cannot use map[string]Int{} (type map[string]Int) as type map[string]map[Int][]string in assignment`

	type F struct{ A []Field }
	// s2 = F{}                   // ERROR `cannot use F{} (type F) as type S in assignment`
	// s2 = struct{ A []Field }{} // ERROR `cannot use struct { A []Field }{} (type struct { A []map[Int][]string }) as type S in assignment`
}

func main() {
	arrayTypes()
	chanTypes()
	definedTypes()
	mapTypes()
	ptrTypes()
	sliceTypes()
	structTypes()
}
