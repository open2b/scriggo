// skip : range over string fails https://github.com/open2b/scriggo/issues/308

// compile

package main

func main() {
	i := 0
	c := ""
	for i, c = range "ab" {
	}
}
