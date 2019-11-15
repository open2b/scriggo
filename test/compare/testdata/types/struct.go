// run

package main

type S1 struct{}
type S2 struct{ A int }
type S3 struct {
	A, B int
	C    string
}

type Int int

type S4 struct {
	A, B Int
}

type S5 struct {
	Int
}

type S6 struct {
	SliceInt []Int
}

type S7 struct {
	Field struct {
		A, B     Int
		SliceInt []Int
		Map      map[Int][]string
	}
}

func main() {
	var s1 S1
	_ = s1

	var s2 S2
	_ = s2
	s2.A = 42

	var s3 S3
	s3.A = s3.B

	// var s4 S4
	// s4.A = 20 // untyped int
	// s4.B = Int(42)

	var s6 S6
	_ = s6

	var s7 S7
	_ = s7

	type Int int
	var _ struct{ A Int } = struct{ A Int }{}
	var _ struct {
		A Int
		B string
	} = struct {
		A Int
		B string
	}{}
}
