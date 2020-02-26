// run

package main

import "fmt"

type T struct{ A int }

func main() {

	// Explicit type.
	e1 := &T{10}
	fmt.Println(e1.A)

	// Slice with elided pointer type.
	s1 := []*T{{20}}
	fmt.Println(s1[0].A)

	// Array with elided pointer type (no ellipsis).
	a1 := [3]*T{{30}}
	fmt.Println(a1[0].A)

	// Array with elided pointer type (with ellipsis).
	a2 := [...]*T{{40}}
	fmt.Println(a2[0].A)

	// Map with pointer element and elided type.
	m1 := map[string]*T{"a": {50}}
	fmt.Println(m1["a"].A)

	// Map with pointer key and elided type.
	m2 := map[*T]string{{60}: "a"}
	_ = m2

	// Map with pointer element and key with elided type on both key and value.
	m3 := map[*T]*T{{70}: {80}}
	_ = m3

}
