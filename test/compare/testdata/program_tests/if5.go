// run

package main

import "fmt"

func main() {
	s := "abc"
	if len(s) < 4 {
		fmt.Print("len(s) < 4\n")
	}
	if len(s) <= 4 {
		fmt.Print("len(s) <= 4\n")
	}
	if len(s) <= 3 {
		fmt.Print("len(s) <= 3\n")
	}
	if len(s) == 3 {
		fmt.Print("len(s) == 3\n")
	}
	if len(s) != 4 {
		fmt.Print("len(s) != 4\n")
	}
	if len(s) >= 3 {
		fmt.Print("len(s) >= 3\n")
	}
	if len(s) >= 2 {
		fmt.Print("len(s) >= 2\n")
	}
	if len(s) > 2 {
		fmt.Print("len(s) > 2\n")
	}

	if 4 > len(s) {
		fmt.Print("4 > len(s)\n")
	}
	if 4 >= len(s) {
		fmt.Print("3 >= len(s)\n")
	}
	if 3 >= len(s) {
		fmt.Print("3 >= len(s)\n")
	}
	if 3 == len(s) {
		fmt.Print("4 == len(s)\n")
	}
	if 4 != len(s) {
		fmt.Print("4 != len(s)\n")
	}
	if 3 <= len(s) {
		fmt.Print("3 <= len(s)\n")
	}
	if 2 <= len(s) {
		fmt.Print("2 <= len(s)\n")
	}
	if 2 < len(s) {
		fmt.Print("2 < len(s)\n")
	}

}
