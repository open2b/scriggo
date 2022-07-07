// run

package main

import "fmt"

type S struct{ I int }

func main() {
	xs := []*S{
		{10}, {20}, &S{30},
	}
	for _, x := range xs {
		fmt.Println((*x).I)
	}
}
