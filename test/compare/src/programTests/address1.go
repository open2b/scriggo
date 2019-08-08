// run

package main

import "fmt"

func main() {

	s1 := "hey"
	s2 := "hoy"
	p1 := &s1
	p2 := &s1
	p3 := &s2
	c1 := p1 == p2
	c2 := p2 == p3
	c3 := p1 == p3
	fmt.Println(c1)
	fmt.Println(c2)
	fmt.Println(c3)

}
