// run

package main

func main() {

	const max8 = 1<<8 - 1
	const max16 = 1<<16 - 1
	const max32 = 1<<32 - 1
	const max64 = 1<<64 - 1

	values8 := []uint8{0, 1, 2, max8 - 2, max8 - 1, max8}

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
	}

	values16 := []uint16{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 2, max16 - 1, max16}

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
	}

	values32 := []uint32{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 2, max32 - 1, max32}

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
	}

	values64 := []uint64{0, 1, 2, max8 - 1, max8, max8 + 1, max16 - 1, max16, max16 + 1, max32 - 1, max32, max32 - 1,
		max64 - 2, max64 - 1, max64}

	for _, i := range values64 {
		for _, j := range values64 {
			println(i, "+", j, "=", i+j)
			println(i, "-", j, "=", i-j)
			println(i, "*", j, "=", i*j)
			if j != 0 {
				println(i, "/", j, "=", i/j)
				println(i, "%", j, "=", i%j)
			}
		}
	}

}
