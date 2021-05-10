// run

package main

func main() {
	for _, v := range [1]int{0} {
		_ = &v
	}
}
