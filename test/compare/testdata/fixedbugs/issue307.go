// skip : continue statement panics https://github.com/open2b/scriggo/issues/178

// compile

package main

func main() {
	for i := 0; i < 5; i++ {
		if i == 4 {
			continue
		}
	}
}
