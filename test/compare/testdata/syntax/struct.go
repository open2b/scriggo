// errorcheck

package main

import p "testpkg"

func main() {

	type T int
	type S string

	type _ struct{}
	type _ struct{,}           // ERROR `unexpected comma, expecting field name or embedded type`
	type _ struct{a int}
	type _ struct{a int,}      // ERROR `unexpected comma, expecting semicolon or newline or }`
	type _ struct{a, b int}
	type _ struct{a, b -}      // ERROR `unexpected -, expecting type`
	type _ struct{a, * }       // ERROR `unexpected *, expecting name`
	type _ struct{T}
	type _ struct{T; S}
	type _ struct{T, S}        // ERROR `unexpected }, expecting type`
	type _ struct{*T}
	type _ struct{*T,}         // ERROR `unexpected comma, expecting semicolon or newline or }`
	type _ struct{&T}          // ERROR `unexpected &, expecting field name or embedded type`
	type _ struct{**T}         // ERROR `unexpected *, expecting name`
	type _ struct{a, b int; S}
	type _ struct{S; a, b int}
	type _ struct{p.T}
	type _ struct{*p.T}
	type _ struct{(T)}         // ERROR `cannot parenthesize embedded type`
	type _ struct{(*T)}        // ERROR `cannot parenthesize embedded type`
	type _ struct{*(T)}        // ERROR `cannot parenthesize embedded type`
	type _ struct{x (int)}
	type _ struct{p.}          // ERROR `unexpected }, expecting name`
	type _ struct{*p.}         // ERROR `syntax error: unexpected }, expecting name`
	type _ struct{p./}         // ERROR `unexpected /, expecting name`
	type _ struct{*p./}        // ERROR `unexpected /, expecting name`
	type _ struct{p.T int}     // ERROR `unexpected int, expecting semicolon or newline or }`
	type _ struct{ map[int]string }  // ERROR `unexpected map, expecting field name or embedded type`
	type _ struct{ []string }  // ERROR `unexpected [, expecting field name or embedded type`

}