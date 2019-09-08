// run

package main

import "fmt"

func triplet() (int, string, string) {
	return 20, "new1", "new2"
}

func main() {
	ss := []string{"old1", "old2", "old3"}
	is := []int{1, 2, 3}
	fmt.Println(ss, is)
	is[0], ss[1], ss[0] = triplet()
	fmt.Println(ss, is)
}
