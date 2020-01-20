// errorcheck

package main

import (
	init "os"  // ERROR `cannot import package as init - init must be a func`
)

var init = 5 // ERROR `cannot declare init - must be func`
var main = 5 // ERROR `cannot declare main - must be func`

var (
	init = 5  // ERROR `cannot declare init - must be func`
	main = 5  // ERROR `cannot declare main - must be func`
)

const init = 5 // ERROR `cannot declare init - must be func`
const main = 5 // ERROR `cannot declare main - must be func`

const (
	init = 5  // ERROR `cannot declare init - must be func`
	main = 5  // ERROR `cannot declare main - must be func`
)

type init int // ERROR `cannot declare init - must be func`
type main int // ERROR `cannot declare main - must be func`

func init() {}
func main() {}
