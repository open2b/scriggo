// run

package main

func F(s []string) {
	for _ = range s {
		return
	}
	return
}

func main() {
	F([]string{"a", "b", "c"})
}
