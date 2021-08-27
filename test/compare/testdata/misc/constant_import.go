// run

package main

import (
	"io"
	"math"
	"time"

	"github.com/open2b/scriggo/test/compare/testpkg"
)

func main() {

	println(math.Pi, math.Pi == 3.14159265358979323846264338327950288419716939937510582097494459)
	println(math.MaxFloat64, math.MaxFloat64 == 1.797693134862315708145274237317043567981e+308)
	println(math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64 == 4.940656458412465441765687928682213723651e-324)
	println(math.MinInt32, math.MinInt64 == -1<<31)
	println(math.MaxInt32, math.MaxInt32 == 1<<32-1)

	println(io.SeekCurrent, io.SeekCurrent == 1)
	println(io.SeekStart, io.SeekStart == 0)

	println(time.UnixDate, time.UnixDate == "Mon Jan _2 15:04:05 MST 2006")

	println(testpkg.C1, testpkg.C1 == "a\t|\"c")
	println(testpkg.C2, testpkg.C2 == true)
	println(testpkg.C3, testpkg.C3 == 1982717381)
	println(testpkg.C4, testpkg.C4 == 1.319382)
	println(testpkg.C5, testpkg.C5 == 3.90i)
	println(testpkg.C6, testpkg.C6 == 'a')
	println(testpkg.C7, testpkg.C7 == 1.0261+2.845i)
	println(testpkg.C8, testpkg.C8 == 1+1.3-'b'+1i)
	println(testpkg.C9, testpkg.C9 == 1+0i)
}
