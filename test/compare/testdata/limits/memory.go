// skip : enable before merging with 'master'.

// paniccheck -mem=10B

package main

func main() {
	s := [10]string{}
	for i := 0; i < 10; i++ {
		s[i] = "stringggg!!!"
	}
}
