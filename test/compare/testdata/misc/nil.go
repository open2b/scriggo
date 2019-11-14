// run

package main

import "fmt"

func main() {

	{
		m1 := map[string]interface{}{}
		m1["k"] = 3
		m1["k2"] = nil
		fmt.Println(m1)
	}

	{
		s1 := []interface{}{0, 0, 0}
		s1[0] = nil
	}

}
