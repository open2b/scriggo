// run

package main

import "fmt"

func main() {

	mess := []interface{}{3, "hey", 5.6, 2}
	for e, v := range mess {
		fmt.Print(e, v, ",")
	}

}
