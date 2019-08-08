// run

package main

import "fmt"

func main() {
	const d = 3e20 / 500000000
	fmt.Printf("d has type %T", d)
	_ = int64(d)
}
