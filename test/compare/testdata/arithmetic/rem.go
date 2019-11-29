// run

package main

func main() {

	{
		const min, max int8 = -1 << 7, 1<<7 - 1
		values := []int8{min, min + 1, min + 2, -2, -1, 0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const min, max int16 = -1 << 15, 1<<15 - 1
		values := []int16{min, min + 1, min + 2, -2, -1, 0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const min, max int32 = -1 << 16, 1<<16 - 1
		values := []int32{min, min + 1, min + 2, -2, -1, 0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const min, max int64 = -1 << 63, 1<<63 - 1
		values := []int64{min, min + 1, min + 2, -2, -1, 0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const max uint8 = 1<<8 - 1
		values := []uint8{0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const max uint8 = 1<<8 - 1
		values := []uint8{0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const max uint16 = 1<<16 - 1
		values := []uint16{0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const max uint32 = 1<<32 - 1
		values := []uint32{0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

	{
		const max uint64 = 1<<64 - 1
		values := []uint64{0, 1, 2, max - 2, max - 1, max}
		for _, i := range values {
			for _, j := range values {
				if j != 0 {
					println(i, j, i%j)
				}
			}
		}
	}

}
