// run

package main

import (
	"fmt"
)

var pv1 struct{}
var pv2 = struct{}{}
var pv3 struct{ A int }
var pv4 struct{ A int } = struct{ A int }{}

var (
	pv5 struct{}
	pv6 = struct{}{}
	pv7 struct{ A int }
	pv8 struct{ A int } = struct{ A int }{}
)

func main() {

	fmt.Println(pv1, pv2, pv3, pv4)
	fmt.Println(pv5, pv6, pv7, pv8)

	var v1 struct{}
	fmt.Println(v1)

	var v2 = struct{}{}
	fmt.Println(v2)

	var v3 struct{ A int }
	fmt.Println(v3)

	var v4 struct{ A int } = struct{ A int }{}
	fmt.Println(v4)

}
