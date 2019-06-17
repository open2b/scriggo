package compiler

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEncodeDecodeFieldIndex(t *testing.T) {
	cases := [][]int{
		[]int{0},
		[]int{20},
		[]int{200},
		[]int{255},
		[]int{0, 0},
		[]int{1, 2, 3},
		[]int{0, 0, 0, 0},
		[]int{11, 200, 254, 0, 0, 1},
		[]int{0, 0, 1, 0},
		[]int{0, 0, 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v", c), func(t *testing.T) {
			encoded := encodeFieldIndex(c)
			decoded := decodeFieldIndex(encoded)
			if !reflect.DeepEqual(c, decoded) {
				t.Fatalf("%v has been encoded to %v (0x%x) and decoded to %v", c, encoded, encoded, decoded)
			}
		})
	}
}
