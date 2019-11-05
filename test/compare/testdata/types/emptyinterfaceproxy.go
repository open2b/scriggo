// run

package main

import "fmt"



func main() {

	{
		var ok bool
		type Int int
		type String string
		var i1 interface{} = Int(4)
		_, ok = i1.(Int)    ; fmt.Println(ok) // true
		_, ok = i1.(String) ; fmt.Println(ok) // false
		_, ok = i1.(int)    ; fmt.Println(ok) // false
	}

}
