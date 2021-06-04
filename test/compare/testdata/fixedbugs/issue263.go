// run

package main

import "fmt"

func F() (f func()) { return }

func main() {
	fmt.Printf("%#v", F())
}
