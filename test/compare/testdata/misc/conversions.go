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

	values8 := []int8{min8, min8 + 1, min8 + 2, -2, -1, 0, 1, 2, max8 - 2, max8 - 1, max8}

	for _, i := range values8 {
		print("int(int8(", i, ")) = ", int(i), "\n")
		print("int8(int8(", i, ")) = ", int8(i), "\n")
		print("int16(int8(", i, ")) = ", int16(i), "\n")
		print("int32(int8(", i, ")) = ", int32(i), "\n")
		print("int64(int8(", i, ")) = ", int64(i), "\n")
		print("uint(int8(", i, ")) = ", uint(i), "\n")
		print("uint8(int8(", i, ")) = ", uint8(i), "\n")
		print("uint16(int8(", i, ")) = ", uint16(i), "\n")
		print("uint32(int8(", i, ")) = ", uint32(i), "\n")
		print("uint64(int8(", i, ")) = ", uint64(i), "\n")
		print("uintptr(int8(", i, ")) = ", uintptr(i), "\n")
	}

	values16 := []int16{min16, min16 + 1, min16 + 2, min8 + 1, min8, min8 - 1, -2, -1, 0, 1, 2,
		max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1, max16 - 2, max16 - 1, max16}

	for _, i := range values16 {
		print("int(int16(", i, ")) = ", int(i), "\n")
		print("int8(int16(", i, ")) = ", int8(i), "\n")
		print("int16(int16(", i, ")) = ", int16(i), "\n")
		print("int32(int16(", i, ")) = ", int32(i), "\n")
		print("int64(int16(", i, ")) = ", int64(i), "\n")
		print("uint(int16(", i, ")) = ", uint(i), "\n")
		print("uint8(int16(", i, ")) = ", uint8(i), "\n")
		print("uint16(int16(", i, ")) = ", uint16(i), "\n")
		print("uint32(int16(", i, ")) = ", uint32(i), "\n")
		print("uint64(int16(", i, ")) = ", uint64(i), "\n")
		print("uintptr(int16(", i, ")) = ", uintptr(i), "\n")
	}

	values32 := []int32{min32, min32 + 1, min32 + 2, min16 + 1, min16, min16 - 1, min8 + 1, min8, min8 - 1,
		-2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1, max16 - 1, max16, max16 + 1,
		max16u - 1, max16u, max16 + 1, max32 - 2, max32 - 1, max32}

	for _, i := range values32 {
		print("int(int32(", i, ")) = ", int(i), "\n")
		print("int8(int32(", i, ")) = ", int8(i), "\n")
		print("int16(int32(", i, ")) = ", int16(i), "\n")
		print("int32(int32(", i, ")) = ", int32(i), "\n")
		print("int64(int32(", i, ")) = ", int64(i), "\n")
		print("uint(int32(", i, ")) = ", uint(i), "\n")
		print("uint8(int32(", i, ")) = ", uint8(i), "\n")
		print("uint16(int32(", i, ")) = ", uint16(i), "\n")
		print("uint32(int32(", i, ")) = ", uint32(i), "\n")
		print("uint64(int32(", i, ")) = ", uint64(i), "\n")
		print("uintptr(int32(", i, ")) = ", uintptr(i), "\n")
	}

	values64 := []int64{min64, min64 + 1, min64 + 2, min32 + 1, min32, min32 - 1, min16 + 1, min16, min16 - 1,
		min8 - 1, min8, min8 + 1, -2, -1, 0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1,
		max16 - 1, max16, max16 + 1, max16u - 1, max16u, max16u + 1, max32 - 1, max32, max32 + 1,
		max32u - 1, max32u, max32u + 1, max64 - 2, max64 - 1, max64}

	for _, i := range values64 {
		print("int(int64(", i, ")) = ", int(i), "\n")
		print("int8(int64(", i, ")) = ", int8(i), "\n")
		print("int16(int64(", i, ")) = ", int16(i), "\n")
		print("int32(int64(", i, ")) = ", int32(i), "\n")
		print("int64(int64(", i, ")) = ", int64(i), "\n")
		print("uint(int64(", i, ")) = ", uint(i), "\n")
		print("uint8(int64(", i, ")) = ", uint8(i), "\n")
		print("uint16(int64(", i, ")) = ", uint16(i), "\n")
		print("uint32(int64(", i, ")) = ", uint32(i), "\n")
		print("uint64(int64(", i, ")) = ", uint64(i), "\n")
		print("uintptr(int64(", i, ")) = ", uintptr(i), "\n")
	}

	values8u := []uint8{0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 2, max8u - 1, max8u}

	for _, i := range values8u {
		print("int(uint8(", i, ")) = ", int(i), "\n")
		print("int8(uint8(", i, ")) = ", int8(i), "\n")
		print("int16(uint8(", i, ")) = ", int16(i), "\n")
		print("int32(uint8(", i, ")) = ", int32(i), "\n")
		print("int64(uint8(", i, ")) = ", int64(i), "\n")
		print("uint(uint8(", i, ")) = ", uint(i), "\n")
		print("uint8(uint8(", i, ")) = ", uint8(i), "\n")
		print("uint16(uint8(", i, ")) = ", uint16(i), "\n")
		print("uint32(uint8(", i, ")) = ", uint32(i), "\n")
		print("uint64(uint8(", i, ")) = ", uint64(i), "\n")
		print("uintptr(uint8(", i, ")) = ", uintptr(i), "\n")
	}

	values16u := []uint16{0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1,
		max16 - 1, max16, max16 + 1, max16u - 2, max16u - 1, max16u}

	for _, i := range values16u {
		print("int(uint16(", i, ")) = ", int(i), "\n")
		print("int8(uint16(", i, ")) = ", int8(i), "\n")
		print("int16(uint16(", i, ")) = ", int16(i), "\n")
		print("int32(uint16(", i, ")) = ", int32(i), "\n")
		print("int64(uint16(", i, ")) = ", int64(i), "\n")
		print("uint(uint16(", i, ")) = ", uint(i), "\n")
		print("uint8(uint16(", i, ")) = ", uint8(i), "\n")
		print("uint16(uint16(", i, ")) = ", uint16(i), "\n")
		print("uint32(uint16(", i, ")) = ", uint32(i), "\n")
		print("uint64(uint16(", i, ")) = ", uint64(i), "\n")
		print("uintptr(uint16(", i, ")) = ", uintptr(i), "\n")
	}

	values32u := []uint32{0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1,
		max16 - 1, max16, max16 + 1, max16u - 1, max16u, max16u + 1,
		max32 - 1, max32, max32 + 1, max32u - 2, max32u - 1, max32u}

	for _, i := range values32u {
		print("int(uint32(", i, ")) = ", int(i), "\n")
		print("int8(uint32(", i, ")) = ", int8(i), "\n")
		print("int16(uint32(", i, ")) = ", int16(i), "\n")
		print("int32(uint32(", i, ")) = ", int32(i), "\n")
		print("int64(uint32(", i, ")) = ", int64(i), "\n")
		print("uint(uint32(", i, ")) = ", uint(i), "\n")
		print("uint8(uint32(", i, ")) = ", uint8(i), "\n")
		print("uint16(uint32(", i, ")) = ", uint16(i), "\n")
		print("uint32(uint32(", i, ")) = ", uint32(i), "\n")
		print("uint64(uint32(", i, ")) = ", uint64(i), "\n")
		print("uintptr(uint32(", i, ")) = ", uintptr(i), "\n")
	}

	values64u := []uint64{0, 1, 2, max8 - 1, max8, max8 + 1, max8u - 1, max8u, max8u + 1,
		max16 - 1, max16, max16 + 1, max16u - 1, max16u, max16u + 1, max32 - 1, max32, max32 + 1,
		max32u - 1, max32u, max32u + 1, max64u - 2, max64u - 1, max64u}

	for _, i := range values64u {
		print("int(uint64(", i, "u)) = ", int(i), "\n")
		print("int8(uint64(", i, ")) = ", int8(i), "\n")
		print("int16(uint64(", i, ")) = ", int16(i), "\n")
		print("int32(uint64(", i, ")) = ", int32(i), "\n")
		print("int64(uint64(", i, ")) = ", int64(i), "\n")
		print("uint(uint64(", i, ")) = ", uint(i), "\n")
		print("uint8(uint64(", i, ")) = ", uint8(i), "\n")
		print("uint16(uint64(", i, ")) = ", uint16(i), "\n")
		print("uint32(uint64(", i, ")) = ", uint32(i), "\n")
		print("uint64(uint64(", i, ")) = ", uint64(i), "\n")
		print("uintptr(uint64(", i, ")) = ", uintptr(i), "\n")
	}
}
