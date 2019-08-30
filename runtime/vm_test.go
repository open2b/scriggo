package runtime

import (
	"testing"
)

type expectedSwapStack struct {
	a, b [4]uint32
	regs registers
}

var swapStackTests = []struct {
	a, b     [4]uint32
	bSize    StackShift
	regs     registers
	expected expectedSwapStack
}{
	// | 0 |  =>  | 0 |
	//   ^          ^
	//  a,b        b,a
	{},

	// | 0 | 3 |  =>  | 0 | 3 |
	//   ^              ^   ^
	//  a,b             b   a
	{
		bSize: StackShift{1},
		regs:  registers{int: []int64{0, 3}},
		expected: expectedSwapStack{
			a:    [4]uint32{1},
			regs: registers{int: []int64{0, 3}},
		},
	},

	// | 0 | 5 |  =>  | 0 | 5 |
	//   ^   ^          ^
	//   a   b         b,a
	{
		regs: registers{int: []int64{0, 5}},
		b:    [4]uint32{1},
		expected: expectedSwapStack{
			regs: registers{int: []int64{0, 5}},
		},
	},

	// | 0 | 5 | 3 |  =>  | 0 | 3 | 5 |
	//   ^   ^              ^   ^
	//   a   b              b   a
	{
		b:     [4]uint32{1},
		bSize: StackShift{1},
		regs:  registers{int: []int64{0, 5, 3}},
		expected: expectedSwapStack{
			a:    [4]uint32{1},
			regs: registers{int: []int64{0, 3, 5}},
		},
	},

	// | 0 | 5 | 8 | 2 | 3 | 6 | 1 |  =>  | 0 | 3 | 6 | 1 | 5 | 8 | 2 |
	//   ^           ^                      ^           ^
	//   a           b                      b           a
	{
		b:     [4]uint32{3},
		bSize: StackShift{3},
		regs:  registers{int: []int64{0, 5, 8, 2, 3, 6, 1}},
		expected: expectedSwapStack{
			a:    [4]uint32{3},
			regs: registers{int: []int64{0, 3, 6, 1, 5, 8, 2}},
		},
	},

	// | 0 | 6 | 0 | 1 | 3 | 6 | 1 | 7 |  =>  | 0 | 6 | 0 | 6 | 1 | 7 | 1 | 3 |
	//           ^       ^                              ^           ^
	//           a       b                              b           a
	{
		a:     [4]uint32{2},
		b:     [4]uint32{4},
		bSize: StackShift{3},
		regs:  registers{int: []int64{0, 6, 0, 1, 3, 6, 1, 7}},
		expected: expectedSwapStack{
			a:    [4]uint32{5},
			b:    [4]uint32{2},
			regs: registers{int: []int64{0, 6, 0, 6, 1, 7, 1, 3}},
		},
	},

	// | 0.0 | 6.0 | 0.0 | 1.0 | 3.0 | 6.0 | 1.0 | 7.0 |  =>  | 0.0 | 6.0 | 0.0 | 6.0 | 1.0 | 7.0 | 1.0 | 3.0 |
	//                ^           ^                                          ^                 ^
	//                a           b                                          b                 a
	{
		a:     [4]uint32{0, 2},
		b:     [4]uint32{0, 4},
		bSize: StackShift{0, 3},
		regs:  registers{float: []float64{0, 6, 0, 1, 3, 6, 1, 7}},
		expected: expectedSwapStack{
			a:    [4]uint32{0, 5},
			b:    [4]uint32{0, 2},
			regs: registers{float: []float64{0, 6, 0, 6, 1, 7, 1, 3}},
		},
	},

	// | "" | "e" | "c" | "s" | "e" | "h" | "p" | "d" |  =>  | "" | "e" | "c" | "h" | "p" | "d" | "s" | "e" |
	//               ^           ^                                         ^                 ^
	//               a           b                                         b                 a
	{
		a:     [4]uint32{0, 0, 2},
		b:     [4]uint32{0, 0, 4},
		bSize: StackShift{0, 0, 3},
		regs:  registers{string: []string{"", "e", "c", "s", "e", "h", "p", "d"}},
		expected: expectedSwapStack{
			a:    [4]uint32{0, 0, 5},
			b:    [4]uint32{0, 0, 2},
			regs: registers{string: []string{"", "e", "c", "h", "p", "d", "s", "e"}},
		},
	},

	// | nil | 5 | "c" | true | 9 | 7.2 | "s" | nil |  =>  | nil | 5 | "c" | 7.2 | "s" | nil | true | 9 |
	//              ^           ^                                       ^                 ^
	//              a           b                                       b                 a
	{
		a:     [4]uint32{0, 0, 0, 2},
		b:     [4]uint32{0, 0, 0, 4},
		bSize: StackShift{0, 0, 0, 3},
		regs:  registers{general: []interface{}{nil, 5, "c", true, 9, 7.2, "s", nil}},
		expected: expectedSwapStack{
			a:    [4]uint32{0, 0, 0, 5},
			b:    [4]uint32{0, 0, 0, 2},
			regs: registers{general: []interface{}{nil, 5, "c", 7.2, "s", nil, true, 9}},
		},
	},
}

func TestSwapStack(t *testing.T) {

	for n, sst := range swapStackTests {
		m := create(nil)
		for i := 0; i < len(sst.regs.int); i++ {
			m.regs.int[i] = sst.regs.int[i]
		}
		for i := 0; i < len(sst.regs.float); i++ {
			m.regs.float[i] = sst.regs.float[i]
		}
		for i := 0; i < len(sst.regs.string); i++ {
			m.regs.string[i] = sst.regs.string[i]
		}
		for i := 0; i < len(sst.regs.general); i++ {
			m.regs.general[i] = sst.regs.general[i]
		}
		m.swapStack(&sst.a, &sst.b, sst.bSize)
		for i := 0; i < 4; i++ {
			if sst.a[i] != sst.expected.a[i] {
				t.Fatalf("test %d: expected %d for a[%d], got %d", n, sst.expected.a[i], i, sst.a[i])
			}
			if sst.b[i] != sst.expected.b[i] {
				t.Fatalf("test %d: expected %d for b[%d], got %d", n, sst.expected.b[i], i, sst.b[i])
			}
		}
		for i, v := range sst.expected.regs.int {
			if v != m.regs.int[i] {
				t.Fatalf("test %d: expected %d for int[%d], got %d", n, v, i, m.regs.int[i])
			}
		}
		for i, v := range sst.expected.regs.float {
			if v != m.regs.float[i] {
				t.Fatalf("test %d: expected %f for float[%d], got %f", n, v, i, m.regs.float[i])
			}
		}
		for i, v := range sst.expected.regs.string {
			if v != m.regs.string[i] {
				t.Fatalf("test %d: expected %q for string[%d], got %q", n, v, i, m.regs.string[i])
			}
		}
		for i, v := range sst.expected.regs.general {
			if v != m.regs.general[i] {
				t.Fatalf("test %d: expected %#v for general[%d], got %#v", n, v, i, m.regs.general[i])
			}
		}
	}

}
