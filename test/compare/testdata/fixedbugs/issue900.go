// run

package main

import (
	"fmt"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func triple() (string, int, int) {
	return "x", 1, 2
}

func main() {

	t := testpkg.T900{}
	t.M("a")

	if got := t.Len("a"); got != 0 {
		panic(fmt.Sprintf("expected 0, got %d", got))
	}
	if got := t.Len("b", 7); got != 1 {
		panic(fmt.Sprintf("expected 1, got %d", got))
	}
	values := []int{1, 2, 3}
	if got := t.Len("c", values...); got != 3 {
		panic(fmt.Sprintf("expected 3, got %d", got))
	}
	if got := t.Len(triple()); got != 2 {
		panic(fmt.Sprintf("expected 2, got %d", got))
	}

	l := t.Len
	if got := l("a"); got != 0 {
		panic(fmt.Sprintf("expected 0, got %d", got))
	}
	if got := l("b", 7); got != 1 {
		panic(fmt.Sprintf("expected 1, got %d", got))
	}
	values = []int{1, 2, 3}
	if got := l("c", values...); got != 3 {
		panic(fmt.Sprintf("expected 3, got %d", got))
	}
	if got := l(triple()); got != 2 {
		panic(fmt.Sprintf("expected 2, got %d", got))
	}

}
