// errorcheck

package main

type Bool bool
type String string

func main() {

	var a = "a"

	switch { }
	switch { case true: }
	switch { case Bool(true): } // ERROR `invalid case Bool(true) in switch (mismatched types Bool and bool)`
	switch true { }
	switch true { case true: }
	switch true { case Bool(true): } // ERROR `invalid case Bool(true) in switch on true (mismatched types Bool and bool)`
	switch nil { } // ERROR `use of untyped nil`
	switch 5.4 { case float32(5.4): } // ERROR `invalid case float32(5.4) in switch on 5.4 (mismatched types float32 and float64)`
	switch 5 { case 3.8: } // ERROR `constant 3.8 truncated to integer`
	switch 5 { case 'a': }
	switch 'a' { case 5: }
	switch 5.4 { case 2i: } // ERROR `constant 2i truncated to real`
	switch 1 < 2 { case true: }
	switch a < "b" { case true: }
	switch a { case "b": }
	switch a { case String("b"): } // ERROR `invalid case String("b") in switch on a (mismatched types String and string)`

	switch Bool(true) { case false: }
	switch int8(5) { case 12: }
	switch 12i { case 1.2: }
	switch { case 1 < 2: }
	switch { case a < "b": }
	switch Bool(true) { case 1 < 2: }
	switch Bool(true) { case a < "b": }
	switch Bool(true) { case true: }
	switch Bool(true) { case bool(true): } // ERROR `invalid case bool(true) in switch on Bool(true) (mismatched types bool and Bool)`
	switch "a" { case a: }
	switch 5 { case a: } // ERROR `invalid case a in switch on 5 (mismatched types string and int)`
	switch { case nil: } // ERROR `cannot convert nil to type bool`
	switch []int{} { case nil: }

	var b interface{}
	var f func()

	switch b { }
	switch b { case nil: }
	switch b { case 5: }
	switch b { case 1 < 2: }
	switch b { case a < "b": }
	switch b { case []int{}: } // ERROR `invalid case []int literal (type []int) in switch (incomparable type)`
	switch []int{} { case []int{}: } // ERROR `invalid case []int literal in switch (can only compare slice []int literal to nil)`
	switch b { case b: }
	switch f { case f: } // ERROR `invalid case f in switch (can only compare func f to nil)`
	switch f { case nil: }
	switch f { case b: } // ERROR `invalid case b in switch (can only compare func f to nil)`

	switch (struct{}{}) { case struct{}{}: }
	switch (struct{}{}) { case nil: } // ERROR `cannot convert nil to type struct {}`
	switch (&struct{}{}) { case nil: }

}
