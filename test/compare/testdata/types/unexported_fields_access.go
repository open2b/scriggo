// errorcheck

package main

import "time"

var _ = time.Now

func main() {

	{
		// time.Time is a struct type declared in a native package, so its
		// unexported fields cannot be accessed.

		_ = time.Now().loc // ERROR `time.Now().loc undefined (cannot refer to unexported field or method loc)`
	}

	{
		var s struct {
			a int
		}
		// s.a should be accessible because "a" is declared in this package.
		_ = s.a
	}

	{
		type t struct {
			a int
		}
		var s t
		// s.a should be accessible because the type t has an underlying struct
		// type that is declared in this package.
		_ = s.a
	}

}
