// run

package main

type R struct {
	r int
}
type T int

func main() {

	var b interface{} = false
	if b == true {
		println(1)
	}
	if b != false {
		println(2)
	}

	var i interface{} = 7
	if i == 5 {
		println(3)
	}
	if i != 5 {
		println(4)
	}

	var s interface{} = "a"
	if s == "b" {
		println(5)
	}
	if s != "a" {
		println(6)
	}

	var f interface{} = 5.5
	if f == 3.2 {
		println(7)
	}
	if f != 5.5 {
		println(8)
	}

	if true == b {
		println(9)
	}
	if false != b {
		println(10)
	}

	var ii interface{} = 2
	var ii2 = 2
	var ii3 = 3
	if ii == ii3 {
		println(11)
	}
	if ii != ii2 {
		println(12)
	}

	var tt interface{} = T(1)
	var tt1 = T(1)
	var tt2 = T(2)
	if tt == tt2 {
		println(13)
	}
	if tt1 != tt {
		println(14)
	}
	if tt2 == tt {
		println(15)
	}
	if tt != tt1 {
		println(16)
	}

	//var rr interface{} = R{1}
	//var rr1 = R{1}
	//var rr2 = R{2}
	//if rr == rr2 {
	//	println(17)
	//}
	//if rr != rr1 {
	//	println(18)
	//}

}
