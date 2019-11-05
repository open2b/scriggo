// compile

package main

type Key int
type Elem string

var _ = map[Key]Elem{}
var _ = map[int]Elem{}
var _ = map[Key]string{}

type MapIntString map[int]string

type MapIntSliceString map[int][]string

type MapKeyElem map[Key]Elem

func main() {

}
