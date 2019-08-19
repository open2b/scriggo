// skip : https://github.com/open2b/scriggo/issues/298

// run

package main

var s = []struct{ Field interface{} }{{0}}

func f(x interface{}) interface{} {
	switch x := x.(type) {
	case string:
		return x
	}
	return 0
}

func main() {
	for _, c := range s {
		_ = f(c.Field)
	}
}
