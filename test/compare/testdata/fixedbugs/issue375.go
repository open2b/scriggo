// run

package main

import (
	"fmt"
	"sort"
)

func main() {

	// Creating a sort.IntSlice.
	is := sort.IntSlice([]int{1, 2, 3})
	fmt.Printf("%T\n", is)

	// Print the length (method and receiver are both non-pointers).
	fmt.Println(is.Len())

	// Take the pointer and create a *sort.IntSlice.
	ptr := &is
	fmt.Println(ptr)
	fmt.Printf("%T\n", ptr)

	// Print the length (method is non-pointer but receiver is a pointer).
	fmt.Println(ptr.Len())

	// Sorting: as before, method is non-pointer but receiver is a pointer.
	ptr.Sort()
	fmt.Println(ptr)
}
