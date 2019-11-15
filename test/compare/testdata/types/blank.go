// errorcheck

package main

type _ int
type _ map[string][]int
type Int int
type _ Int

type _ = int
type _ = map[string][]int
type Int2 = int
type _ = Int

type (
	_  int
	_  = int
	T  struct{ A, B Int }
	_  = T
	T2 T
)

type _ badType // ERROR `undefined: badType`

func main() {

	type _ int
	type _ map[string][]int
	type Int int
	type _ Int

	type _ = int
	type _ = map[string][]int
	type Int2 = int
	type _ = Int

	type (
		_  int
		_  = int
		T  struct{ A, B Int }
		_  = T
		T2 T
	)

	type _ badType // ERROR `undefined: badType`

}
