// compile

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
}
