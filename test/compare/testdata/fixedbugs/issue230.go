// compile

package main

func main() {
	var x interface{}
	switch x := x.(type) {
	case float32:
		_ = x
	default:
	}
}
