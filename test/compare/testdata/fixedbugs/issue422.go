// errorcheck

package main

func main() {
	var _ = map[string]string{"0"} // ERROR `missing key in map literal`
}
