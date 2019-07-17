package main

import (
	"bytes"
	"fmt"
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
}
