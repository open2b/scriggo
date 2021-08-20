// errorcheck

package main

func main() {
	var a struct{ _ func() }
	switch a { } // ERROR `cannot switch on a (struct { 𝗽0 func() } is not comparable)`
	var b [1]struct{ _ func() }
	switch b { } // ERROR `cannot switch on b ([1]struct { 𝗽0 func() } is not comparable)`
	_, _ = a, b
}
