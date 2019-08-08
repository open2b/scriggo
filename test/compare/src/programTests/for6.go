// run

package main

import "fmt"

func main() {

	m := map[string]int{"twelve": 12}
	for k, v := range m {
		fmt.Print(k, " is ", v, ",")
	}

}
