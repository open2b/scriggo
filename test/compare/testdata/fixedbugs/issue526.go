// run

package main

import "fmt"

type Int int

type S struct {
	F interface{}
}

func main() {
	s := S{[]Int{}}
	_, ok := s.F.([]Int)
	fmt.Println(ok)
}
