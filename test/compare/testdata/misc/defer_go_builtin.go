// run

package main

import "fmt"

func main() {

	// Builtin 'close'.

	ch := make(chan int)
	defer close(ch)
	defer close(make(chan int))
	ch2 := make(chan int)
	go close(ch2)
	go close(make(chan string))

	// Builtin 'delete'.

	m := map[string]string{"a": "A"}
	fmt.Println(m)
	delete(m, "a")
	fmt.Println(m)
	delete(map[int]int{4: 16, 20: 400}, 20)

	// // Builtin 'print'.
	// See https://github.com/open2b/scriggo/issues/285#issuecomment-520437921

	// defer print("print")
	// defer print(1, 2, 3)
	// defer print()

	// // Builtin 'println'.

	// defer println("println")
	// defer println(4, 5, 6)
	// defer println()
}
