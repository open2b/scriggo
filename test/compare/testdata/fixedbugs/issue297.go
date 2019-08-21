// compile

package main

var s1 struct{}
var s2 struct{ a int }
var s3 struct{ a int; }
var s4 struct { a int
}
var s5 struct { a int;
}
var s6 struct {
	a int
}
var s7 struct {
	a int;
}

var s8 struct{ int }
var s9 struct{ int; }
var s10 struct { int
}
var s11 struct { int;
}
var s12 struct {
	int
}
var s13 struct {
	int;
}

var s14 struct { a string; b int }
var s15 struct { a string; b int; }
var s16 struct { a string; b int
}
var s17 struct { a string; b int;
}
var s18 struct { a string
	b int
}
var s19 struct { a string
	b int;
}

func main() {

	var a int
	_ = a

	{ }
	{ a = 5 }
	{ a = 5; }
	{ a = 5
	}
	{ a = 5;
	}
	{
		a = 5
	}
	{
		a = 5;
	}

	_ = []int{}
	_ = []int{5}
	_ = []int{5, }
	_ = []int{5, 6}
	_ = []int{5,
	}
	_ = []int{5, 6,
	}
	_ = []int{
		5,
	}
	_ = []int{
		5, 6,
	}

	_ = [2]int{0: 0}
	_ = [2]int{0: 0,
	}
	_ = [2]int{0:
		0,
	}
	_ = [2]int{
		0: 0,
	}
	_ = [2]int{0: 0, 1: 1}
	_ = [2]int{0: 0, 1: 1,
    }
	_ = [2]int{0: 0, 1:
		1,
	}
	_ = [2]int{ 0: 0,
		1: 1,
	}

	_ = map[int]int{0: 0}
	_ = map[int]int{0: 0,
	}
	_ = map[int]int{0:
		0,
	}
	_ = map[int]int{
		0: 0,
	}
	_ = map[int]int{0: 0, 1: 1}
	_ = map[int]int{0: 0, 1: 1,
	}
	_ = map[int]int{0: 0, 1:
		1,
	}
	_ = map[int]int{ 0: 0,
		1: 1,
	}

}
