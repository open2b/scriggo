// compile

package main

func main() {

	type T1 = int
	type T2 = int

	type ( )
	type ( T3 = int )
	type ( T4 = int; )
	type ( T5 = int; T6 = int )
	type (
		T7 = int
		T8 = int
	)
	type (
		T9 = int;
		T10 = int;
	)

}
