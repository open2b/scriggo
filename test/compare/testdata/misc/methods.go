// skip : enable before merging with 'master'.

// run

package main

import (
	"bytes"
	"fmt"
	"sort"
	"time"
)

func main() {

	//---------------------------------------------------------------------------
	// Method calls on concrete values
	//---------------------------------------------------------------------------

	// Receiver has type    *T
	// Method is defined on *T
	//
	{
		b := bytes.NewBufferString("a")
		s := b.String()
		fmt.Print(s)
	}

	// Receiver has type    T
	// Method is defined on *T
	//
	// b.String() is equivalent to (&b).String()
	{
		var b = *bytes.NewBufferString("b")
		s := b.String()
		fmt.Print(s)
	}

	// Receiver has type T
	// Method is defined on T
	//
	{
		var d time.Duration
		d += 7200000000000
		h := d.Hours()
		fmt.Print(h)
	}

	// Receiver has type *T
	// Method is defined on T
	//
	// dp.Seconds() is equivalent to (*dp).Seconds()
	{
		var d time.Duration
		dp := &d
		d = 125
		h := dp.Nanoseconds()
		fmt.Print(h)
	}

	{
		// Create a struct of struct and call a method on it;
		// the method must be defined on non-pointer.

		type S = struct {
			Field sort.IntSlice
		}

		s := S{
			Field: sort.IntSlice([]int{5, 6, 1}),
		}

		fmt.Printf("%v %T\n", s, s)

		s.Field.Sort()

		fmt.Printf("%v %T\n", s, s)
	}

}
