// run

package main

type key struct {
	X int
	Y int
}

func main() {
	println("starting population")
	mpbasic := map[int]int{}
	mpcomplex := map[key]int{}
	for i := 0; i < 256; i++ {
		mpbasic[i] = i * 2
		mpcomplex[key{X: i, Y: i + 1}] = i * 2
	}
	println("starting delete simple")
	for k := range mpbasic {
		delete(mpbasic, k)
	}
	println("starting delete complex")
	for k := range mpcomplex {
		if mpcomplex[k] > 2 {
			delete(mpcomplex, k)
		}
	}
	println("done")
}
