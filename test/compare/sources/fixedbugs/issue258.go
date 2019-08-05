// skip : https://github.com/open2b/scriggo/issues/258

// compile

package main

func main() {
	var _ = nil == interface{}(nil)
}