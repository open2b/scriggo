// run

package main

import "fmt"



func main() {

	{

		// Defined types.

		var ok bool
		
		type Int int
		type String string
		type SliceByte []byte

		var i1 interface{} = Int(4)
		_, ok = i1.(Int)    ; fmt.Println(ok) // true
		_, ok = i1.(String) ; fmt.Println(ok) // false
		_, ok = i1.(int)    ; fmt.Println(ok) // false

		var i2 interface{} = String("hello")
		_, ok = i2.(Int)    ; fmt.Println(ok) // false
		_, ok = i2.(String) ; fmt.Println(ok) // true
		_, ok = i2.(int)    ; fmt.Println(ok) // false

		var i3 interface{} = SliceByte([]byte{})
		_, ok = i3.(SliceByte) ; fmt.Println(ok) // true
		_, ok = i3.([]byte)    ; fmt.Println(ok) // false
	}

	{
		// Slice types.

		var ok bool

		type Int int
		var i1 interface{} = []Int{1,2,3}
		_ , ok = i1.(Int)   ; fmt.Println(ok)
		_ , ok = i1.([]int) ; fmt.Println(ok)
		_ , ok = i1.([]Int) ; fmt.Println(ok)

	}

	{

		// Map types.

		var ok bool

		type Int int
		type String string

		var i1 interface{} = map[String]Int{"ciao": 3}
		_ , ok = i1.(Int)            ; fmt.Println(ok)
		_ , ok = i1.(map[string]int) ; fmt.Println(ok)
		_ , ok = i1.(map[String]Int) ; fmt.Println(ok)

	}

	{
		// Struct types.

		var ok bool

		type (
			Int int
			String string
		)

		var i1 interface{} = struct{A Int}{42}
		_, ok = i1.(Int); fmt.Println(ok)
		_, ok = i1.(struct{A int}); fmt.Println(ok)
		// _, ok = i1.(struct{A Int}); fmt.Println(ok) // TODO: cannot pass until structType has a pointer inside of it.
	}

	{
		// Indirect values.

		var ok bool

		type Int int

		i := Int(42)
		_ = &i
		var interf interface{} = i
		_, ok = interf.(Int)  ; fmt.Println(ok)
		_, ok = interf.(int)  ; fmt.Println(ok)
		_, ok = interf.(*int) ; fmt.Println(ok)
		_, ok = interf.(*Int) ; fmt.Println(ok)
	}


}
