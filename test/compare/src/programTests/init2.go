// run

package main

import (
	"fmt"
)

func init() {
	fmt.Print("init1!")
}

func main() {
	fmt.Print("main!")
}

func init() {
	fmt.Print("init2!")
}

func init() {
	fmt.Print("init3!")
}
