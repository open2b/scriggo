// run

package main

import "fmt"

func main() {
	switch u := interface{}("test").(type) {
	case string:
		fmt.Print("string is: ")
		fmt.Print(u)
	}
}
