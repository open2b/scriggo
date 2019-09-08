// run

package main

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out = out + s
	}
	return out
}

func main() {
	var b, d int
	var s1, f, s2 string

	b = 1
	d = b + 2
	s1 = repeat("z", 4)
	f = "hello"
	s2 = repeat(f, d)

	_ = s1 + s2
	return
}
