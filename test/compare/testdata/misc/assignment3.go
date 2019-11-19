// errorcheck

package main

func main() {

	{
		a := 10
		_ = a
		a, b := "string", 20 ; _ = b // ERROR `cannot use "string" (type string) as type int in assignment`		
	}

}
