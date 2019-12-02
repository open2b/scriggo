// run

package main

func F1(_, _ string)               {}
func F2() (_, _ string)            { return "", "" }
func F3(_, _ string) (_, _ string) { return "", "" }
func F4(_, _ string, ints ...int)  {}

func main() {
	_ = func(_, _ string) {}
	_ = func() (_, _ string) { return "", "" }
	_ = func(_, _ string) (_, _ string) { return "", "" }
	_ = func(_, _ string, ints ...int) {}
}
