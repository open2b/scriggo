// run

package main

type I interface{}

type M map[int]int

func main() {

	m := make(map[I]int)
	var key interface{} = 5
	delete(m, key)

	m2 := make(M)
	delete(m2, 5)

}
