// errorcheck

package main

func h(x, y ...int) {} // ERROR `cannot use ... with non-final parameter x`

func main() {}
