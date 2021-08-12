// errorcheck

package main

type T1 []T1 // ERROR `invalid recursive type T`

type T2 = T2 // ERROR `invalid recursive type alias T`

func main() {}
