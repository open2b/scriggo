// errorcheck

package main

func main() {

	i := interface{}(nil); _ = i.({}) // ERROR `syntax error: unexpected {, expecting type`

}