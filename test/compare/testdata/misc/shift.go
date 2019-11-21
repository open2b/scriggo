// run

package main

import "fmt"

func main() {

	{
		n := 1
		var i interface{} = 300 >> n
		fmt.Println(i)
	}

}
