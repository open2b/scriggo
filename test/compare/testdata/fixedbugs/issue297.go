// compile

package main

var s1 struct{}
var s2 struct{ a int }
var s3 struct{ a int; }
var s4 struct{ a int
}
var s5 struct{ a int;
}
var s6 struct{
	a int
}
var s7 struct{
	a int;
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

}
