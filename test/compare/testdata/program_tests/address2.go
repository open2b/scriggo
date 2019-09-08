// run

package main

import "fmt"

func main() {

	var s1 = "hey"
	var s2 = "hey"
	var p1 = &s1
	var p2 = &s1
	var p3 = &s2
	c1 := p1 == p2
	c2 := p2 == p3
	c3 := p1 == p3
	fmt.Println(c1)
	fmt.Println(c2)
	fmt.Println(c3)

}
