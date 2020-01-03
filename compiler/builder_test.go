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

func TestComplex128(t *testing.T) {
	operations := []struct {
		c1, c2, neg, add, sub, mul, div complex128
	}{
		{c1: 0, c2: 0, neg: 0, add: 0, sub: 0, mul: 0},
		{c1: 1, c2: 1, neg: -1, add: 2, sub: 0, mul: 1, div: 1},
		{c1: 1i, c2: 1i, neg: -1i, add: 2i, sub: 0, mul: -1, div: 1},
		{
			c1:  1 + 2i,
			c2:  3 + 5i,
			neg: -1 - 2i,
			add: 4 + 7i,
			sub: -2 - 3i,
			mul: -7 + 11i,
			div: 0.382352941176470617623550651842379011213779449462890625 + 0.02941176470588234559411233703940524719655513763427734375i,
		},
		{
			c1:  23.95 - 93.04i,
			c2:  7.50 + 2i,
			neg: -23.95 + 93.04i,
			add: 31.45 - 91.04i,
			sub: 16.45 - 95.04i,
			mul: 365.7050000000000409272615797817707061767578125 - 649.90000000000009094947017729282379150390625i,
			div: -0.10713692946058138433240713993654935620725154876708984375 - 12.376763485477180637417404795996844768524169921875i,
		},
		{
			c1:  73829571043429.02756423 + 928746285629.7836396i,
			c2:  29470173655.93846244 + 79549687362.927342745i,
			neg: -73829571043429.02756423 - 928746285629.7836396i,
			add: 73859041217084.968750 + 1008295972992.711060i,
			sub: 73800100869773.09375 + 849196598266.856323i,
			mul: 2101888802931970056126464 + 5900489608963630053720064i,
			div: 312.5973424729687621947959996759891510009765625 - 812.288209022923865632037632167339324951171875i,
		},
	}
	for _, op := range operations {
		c := negComplex(op.c1)
		c3, ok := c.(complex128)
		if !ok {
			t.Fatalf("-(%f): unexpected result type %T, expected complex128", op.c1, c)
		}
		if c3 != op.neg {
			t.Fatalf("-(%f): unexpected result %f, expected %f", op.c1, c3, op.neg)
		}
		c = addComplex(op.c1, op.c2)
		c3, ok = c.(complex128)
		if !ok {
			t.Fatalf("%f + %f: unexpected result type %T, expected complex128", op.c1, op.c2, c)
		}
		if c3 != op.add {
			t.Fatalf("%f + %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.add)
		}
		c = subComplex(op.c1, op.c2)
		c3, ok = c.(complex128)
		if !ok {
			t.Fatalf("%f - %f: unexpected result type %T, expected complex128", op.c1, op.c2, c)
		}
		if c3 != op.sub {
			t.Fatalf("%f - %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.sub)
		}
		c = mulComplex(op.c1, op.c2)
		c3, ok = c.(complex128)
		if !ok {
			t.Fatalf("%f * %f: unexpected result type %T, expected complex128", op.c1, op.c2, c)
		}
		if c3 != op.mul {
			t.Fatalf("%f * %f: unexpected result %.54f, expected %.54f", op.c1, op.c2, c3, op.mul)
		}
		if op.c2 != 0 {
			c = divComplex(op.c1, op.c2)
			c3, ok = c.(complex128)
			if !ok {
				t.Fatalf("%f / %f: unexpected result type %T, expected complex128", op.c1, op.c2, c)
			}
			if c3 != op.div {
				t.Fatalf("%f / %f: unexpected result %.60f, expected %.60f", op.c1, op.c2, c3, op.div)
			}
		}
	}
}

func TestComplex64(t *testing.T) {
	operations := []struct {
		c1, c2, neg, add, sub, mul, div complex64
	}{
		{c1: 0, c2: 0, neg: 0, add: 0, sub: 0, mul: 0},
		{c1: 1, c2: 1, neg: -1, add: 2, sub: 0, mul: 1, div: 1},
		{c1: 1i, c2: 1i, neg: -1i, add: 2i, sub: 0, mul: -1, div: 1},
		{
			c1:  1 + 2i,
			c2:  3 + 5i,
			neg: -1 - 2i,
			add: 4 + 7i,
			sub: -2 - 3i,
			mul: -7 + 11i,
			div: 0.38235294818878173828125 + 0.02941176481544971466064453125i,
		},
		{
			c1:  23.95 - 93.04i,
			c2:  7.50 + 2i,
			neg: -23.95 + 93.04i,
			add: 31.45 - 91.04i,
			sub: 16.45 - 95.04i,
			mul: 365.70501708984375 - 649.9000244140625i,
			div: -0.107136867940425872802734375 - 12.37676334381103515625i,
		},
		{
			c1:  7382.02756423 + 92.7836396i,
			c2:  294.9384765625 + 795.9273681640625i,
			neg: -7382.02756423 - 92.7836396i,
			add: 7676.9658203125 + 888.71099853515625i,
			sub: 7087.0888671875 - 703.14373779296875i,
			mul: 2103394.75 + 5902923i,
			div: 3.1243956089019775390625 - 8.1169757843017578125i,
		},
	}
	for _, op := range operations {
		c := negComplex(op.c1)
		c3, ok := c.(complex64)
		if !ok {
			t.Fatalf("-(%f): unexpected result type %T, expected complex64", op.c1, c)
		}
		if c3 != op.neg {
			t.Fatalf("-(%f): unexpected result %f, expected %f", op.c1, c3, op.neg)
		}
		c = addComplex(op.c1, op.c2)
		c3, ok = c.(complex64)
		if !ok {
			t.Fatalf("%f + %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.add {
			t.Fatalf("%f + %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.add)
		}
		c = subComplex(op.c1, op.c2)
		c3, ok = c.(complex64)
		if !ok {
			t.Fatalf("%f - %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.sub {
			t.Fatalf("%f - %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.sub)
		}
		c = mulComplex(op.c1, op.c2)
		c3, ok = c.(complex64)
		if !ok {
			t.Fatalf("%f * %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.mul {
			t.Fatalf("%f * %f: unexpected result %.20f, expected %.20f", op.c1, op.c2, c3, op.mul)
		}
		if op.c2 != 0 {
			c = divComplex(op.c1, op.c2)
			c3, ok = c.(complex64)
			if !ok {
				t.Fatalf("%f / %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
			}
			if c3 != op.div {
				t.Fatalf("%f / %f: unexpected result %.20f, expected %.20f", op.c1, op.c2, c3, op.div)
			}
		}
	}
}

func TestComplexNotPredeclared(t *testing.T) {
	type Complex complex64
	operations := []struct {
		c1, c2, neg, add, sub, mul, div Complex
	}{
		{
			c1:  1 + 2i,
			c2:  3 + 5i,
			neg: -1 - 2i,
			add: 4 + 7i,
			sub: -2 - 3i,
			mul: -7 + 11i,
			div: 0.38235294818878173828125 + 0.02941176481544971466064453125i,
		},
	}
	for _, op := range operations {
		c := negComplex(op.c1)
		c3, ok := c.(Complex)
		if !ok {
			t.Fatalf("-(%f): unexpected result type %T, expected complex64", op.c1, c)
		}
		if c3 != op.neg {
			t.Fatalf("-(%f): unexpected result %f, expected %f", op.c1, c3, op.neg)
		}
		c = addComplex(op.c1, op.c2)
		c3, ok = c.(Complex)
		if !ok {
			t.Fatalf("%f + %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.add {
			t.Fatalf("%f + %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.add)
		}
		c = subComplex(op.c1, op.c2)
		c3, ok = c.(Complex)
		if !ok {
			t.Fatalf("%f - %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.sub {
			t.Fatalf("%f - %f: unexpected result %f, expected %f", op.c1, op.c2, c3, op.sub)
		}
		c = mulComplex(op.c1, op.c2)
		c3, ok = c.(Complex)
		if !ok {
			t.Fatalf("%f * %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
		}
		if c3 != op.mul {
			t.Fatalf("%f * %f: unexpected result %.20f, expected %.20f", op.c1, op.c2, c3, op.mul)
		}
		if op.c2 != 0 {
			c = divComplex(op.c1, op.c2)
			c3, ok = c.(Complex)
			if !ok {
				t.Fatalf("%f / %f: unexpected result type %T, expected complex64", op.c1, op.c2, c)
			}
			if c3 != op.div {
				t.Fatalf("%f / %f: unexpected result %.20f, expected %.20f", op.c1, op.c2, c3, op.div)
			}
		}
	}
}
