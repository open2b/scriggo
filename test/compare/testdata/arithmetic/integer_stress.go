// run

package main

func main() {

	const min8, max8 = -1 << 7, 1<<7 - 1
	const min16, max16 = -1 << 15, 1<<15 - 1
	const min32, max32 = -1 << 16, 1<<16 - 1
	const min64, max64 = -1 << 63, 1<<63 - 1

	const max8u = 1<<8 - 1
	const max16u = 1<<16 - 1
	const max32u = 1<<32 - 1
	const max64u = 1<<64 - 1

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 int8 =
			min8, min8 + 1,  -45, -1, 0, 1, 80, 119, max8 - 1, max8
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 int16 =
			min16, min16 + 1, -450, -1, 0, 1, 803, 11903, max16 - 1, max16
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)

	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 int32 =
			min32, min32 + 1,  -79047667, -1, 0, 1, 20913, 42750183, max32 - 1, max32
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 int64 =
			min64, min64 + 1,  -782738648362440, -1, 0, 1, 18492625473483, 7938372634725374, max64 - 1, max64
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 uint8 =
			0, 1, 39, 107, max8-1, max8, max8 + 1, 199, max8u - 1, max8u
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 uint16 =
			0, 1, 39, 107, max16-1, max16, max16 + 1, 45092, max16u - 1, max16u
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 uint32 =
			0, 1, 5273636, 901837394, max32-1, max32, max32 + 1, 3284730661, max32u - 1, max32u
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}

	{
		var x1, x2, x3, x4, x5, x6, x7, x8, x9, x10 uint64 =
			0, 1, 802938342673, 1840722946571530694, max64-1, max64, max64 + 1, 13507834173395047816, max64u - 1, max64u
		var x = x1 + x4 - x8*x10/(x3-x1%x7) + x3*x1 - x8%x3 +  x5/x9 + x6*x7 + x2
		println(x)
	}
}
