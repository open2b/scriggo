// run

package main

func main() {
	type S struct{ A int }
	s := func() *S {
		return &S{}
	}
	_ = s
}
