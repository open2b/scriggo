// errorcheck

// Do not fmt this file.

package main

func main() {
}

func t1() bool {
	return true
	var _ int // ERROR `missing return`
}

func t2() bool {
	b := false
L:
	if b {
		return true
	}
	b = true
	goto L
	var _ int // ERROR `missing return`
}

func t3() bool {
	panic(true)
	var _ int // ERROR `missing return`
}

func t4() bool {
	{
		return true
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t5() bool {
	return true
	{} // ERROR `missing return`
}

func t6() bool {
	if true {
		return true
		var _ int // ERROR `missing return`
	} else {
		return false
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t7() bool {
	if true {
		return true
		var _ int // ERROR `missing return`
	} else if false {
		return false
		var _ int // ERROR `missing return`
	} else {
		return true
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t8() bool {
	return true
	if true { return true } // ERROR `missing return`
	if true { return true } else if false { return false }  // ERROR `missing return`
}

func t9() bool {
	for {
		break // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t10() bool {
	return true
	for false { } // ERROR `missing return`
}

func t11() bool {
	return true
	for range "" { } // ERROR `missing return`
}

func t12() bool {
	for i := 0; ; i++ {
		if i > 0 {
			break // ERROR `missing return`
			return i == 0
		}
		break // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t13() bool {
	var i int
	switch i {
	case 0:
		fallthrough
	case 1:
		goto L
	L:
		fallthrough
	case 2:
		return true
		var _ int // ERROR `missing return`
	default:
		break // ERROR `missing return`
		return false
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t14() bool {
	return true
	switch { } // ERROR `missing return`
}

func t15() bool {
	ch := make(chan bool, 1)
	select {
	case ch <- true:
		break // ERROR `missing return`
		return true
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t16() bool {
	ch := make(chan bool)
	select {
	case ch <- true:
		return true
	default:
		break // ERROR `missing return`
		return false
		var _ int // ERROR `missing return`
	}
	var _ int // ERROR `missing return`
}

func t17() bool {
	goto L
L:
	return true
	var _ int // ERROR `missing return`
}

func t18() bool {
	return true
	goto L; L: _ = 5 // ERROR `missing return`
}
