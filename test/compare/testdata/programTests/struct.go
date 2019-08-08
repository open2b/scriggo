// run

package main

import "fmt"

func main() {
	s1 := struct{ A interface{} }{"string"}
	fmt.Print(s1)
	s2 := struct{ A interface{} }{[]int{10, 20, 30}}
	fmt.Print(s2)
	s3 := struct{ A interface{} }{10.43}
	fmt.Print(s3)
	s4 := struct{ A interface{} }{-42}
	fmt.Print(s4)
}
