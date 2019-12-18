// run

package main

func main() {

	const min8, max8 = -1 << 7, 1<<7 - 1
	const min16, max16 = -1 << 15, 1<<15 - 1
	const min32, max32 = -1 << 16, 1<<16 - 1
	const min64, max64 = -1 << 63, 1<<63 - 1

	values8 := []int8{min8, min8 + 1, min8 + 2, -2, -1, 0, 1, 2, max8 - 2, max8 - 1, max8}

	for _, i := range values8 {
		for _, j := range values8 {
			println(i, "+", j, "=", i+j)
			println(i, "-", j, "=", i-j)
			println(i, "*", j, "=", i*j)
			if j != 0 {
				println(i, "/", j, "=", i/j)
				println(i, "%", j, "=", i%j)
			}
		}
		for j := 0; j <= 64; j++ {
			println(i, "<<", j, "=", i<<j)
		}
		for j := 0; j <= 64; j++ {
			println(i, ">>", j, "=", i>>j)
		}
	}

	values16 := []int16{min16, min16 + 1, min16 + 2, min8 - 1, min8, min8 + 1, -2, -1, 0, 1, 2,
		max8 - 1, max8, max8 + 1, max16 - 2, max16 - 1, max16}

	for _, i := range values16 {
		for _, j := range values16 {
			println(i, "+", j, "=", i+j)
			println(i, "-", j, "=", i-j)
			println(i, "*", j, "=", i*j)
			if j != 0 {
				println(i, "/", j, "=", i/j)
				println(i, "%", j, "=", i%j)
			}
		}
		for j := 0; j <= 64; j++ {
			println(i, "<<", j, "=", i<<j)
		}
		for j := 0; j <= 64; j++ {
			println(i, ">>", j, "=", i>>j)
		}
	}

	values32 := []int32{min32, min32 + 1, min32 + 2, min16 - 1, min16, min16 + 1, min8 - 1, min8, min8 + 1,
		-2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 2, max32 - 1, max32}

	for _, i := range values32 {
		for _, j := range values32 {
			println(i, "+", j, "=", i+j)
			println(i, "-", j, "=", i-j)
			println(i, "*", j, "=", i*j)
			if j != 0 {
				println(i, "/", j, "=", i/j)
				println(i, "%", j, "=", i%j)
			}
		}
		for j := 0; j <= 64; j++ {
			println(i, "<<", j, "=", i<<j)
		}
		for j := 0; j <= 64; j++ {
			println(i, ">>", j, "=", i>>j)
		}
	}

	values64 := []int64{min64, min64 + 1, min64 + 2, min32 - 1, min32, min32 + 1, min16 - 1, min16, min16 + 1,
		min8 - 1, min8, min8 + 1, -2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1,
		max32 - 1, max32, max32 + 1, max64 - 2, max64 - 1, max64}

	for _, i := range values64 {
		for _, j := range values64 {
			println(i, "+", j, "=", i+j)
			println(i, "+", j, "=", i+j)
			println(i, "-", j, "=", i-j)
			println(i, "*", j, "=", i*j)
			if j != 0 {
				println(i, "/", j, "=", i/j)
				println(i, "%", j, "=", i%j)
			}
		}
		for j := 0; j <= 64; j++ {
			println(i, "<<", j, "=", i<<j)
		}
		for j := 0; j <= 64; j++ {
			println(i, ">>", j, "=", i>>j)
		}
	}

}
