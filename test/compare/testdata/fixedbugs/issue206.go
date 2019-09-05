// errorcheck

package main

func len(s interface{}) int { return 0 }

const _ = len([1]int{}) // ERROR `const initializer len([1]int literal) is not a constant`

func main() {}
