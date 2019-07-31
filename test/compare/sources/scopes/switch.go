// runcmp

package main

import "fmt"

func main() {
	a := "ext"
	switch a := 3; a {
	case 3:
		fmt.Println(a)
		a := "internal"
		fmt.Println(a)
	}
	fmt.Println(a)

}
