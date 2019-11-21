// run

package main

import "fmt"

func main() {

	{
		n := 1
		var i interface{} = 300 >> n
		fmt.Println(i)
	}

	{
		r := int(20)
		var s int32 = 1 << r
		_ = s
	}

}
