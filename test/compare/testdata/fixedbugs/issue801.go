// build

package main

type Byte byte

type Slicebyte []byte

type SliceByte []Byte

func main() {
	_ = string([]byte{})
	_ = string([]Byte{})
	_ = string(Slicebyte{})
	_ = string(SliceByte{})

	_ = []byte("")
	_ = []Byte("")
	_ = Slicebyte("")
	_ = SliceByte("")
}
