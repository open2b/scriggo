//
// Copyright (c) 2017 Open2b Software Snc. All Rights Reserved.
//

package types

import (
	"strconv"
	"strings"

	"open2b/decimal"
)

const maxUint = ^uint(0)
const maxInt = int(maxUint >> 1)
const minInt = -maxInt - 1

type Number struct {
	i       int
	d       decimal.Dec
	integer bool
}

func NewNumberInt(i int) Number {
	return Number{i: i, integer: true}
}

func NewNumberString(s string) Number {
	n, err := strconv.ParseInt(s, 10, strconv.IntSize)
	if err == nil {
		return Number{i: int(n), integer: true}
	}
	return Number{d: decimal.String(s)}
}

func (n1 Number) Compared(n2 Number) int {
	if n1.integer {
		if n2.integer {
			if n1.i < n2.i {
				return -1
			} else if n1.i > n2.i {
				return 1
			}
			return 0
		} else {
			return decimal.Int(n1.i).ComparedTo(n2.d)
		}
	} else {
		if n2.integer {
			return n1.d.ComparedTo(decimal.Int(n2.i))
		} else {
			return n1.d.ComparedTo(n2.d)
		}
	}
}

func (n1 Number) Divided(n2 Number) Number {
	if n1.integer {
		if n2.integer {
			if n2.i == 0 {
				panic("template: zero division")
			}
			if n1.i%n2.i == 0 && !(n1.i == minInt && n2.i == -1) {
				return Number{i: n1.i / n2.i, integer: true}
			}
			return Number{d: decimal.Int(n1.i).Divided(decimal.Int(n2.i))}
		} else {
			if n2.d.IsZero() {
				panic("template: zero division")
			}
			return Number{d: decimal.Int(n1.i).Divided(n2.d)}
		}
	} else {
		if n2.integer {
			if n2.i == 0 {
				panic("template: zero division")
			}
			return Number{d: n1.d.Divided(decimal.Int(n2.i))}
		} else {
			if n2.d.IsZero() {
				panic("template: zero division")
			}
			return Number{d: n1.d.Divided(n2.d)}
		}
	}
}

func (n Number) Int() (int, bool) {
	if n.integer {
		return n.i, true
	}
	if n.d.Digits() > 0 {
		return 0, false
	}
	if n.d.ComparedTo(decimal.Int(maxInt)) > 0 {
		return 0, false
	}
	if n.d.ComparedTo(decimal.Int(minInt)) < 0 {
		return 0, false
	}
	i, err := strconv.ParseInt(n.d.String(), 10, strconv.IntSize)
	return int(i), err == nil
}

func (n1 Number) Minus(n2 Number) Number {
	if n1.integer {
		if n2.integer {
			s := n1.i - n2.i
			if (s < n1.i) != (n2.i > 0) {
				return Number{d: decimal.Int(n1.i).Minus(decimal.Int(n2.i))}
			}
			return Number{i: s, integer: true}
		} else {
			return Number{d: decimal.Int(n1.i).Minus(n2.d)}
		}
	} else {
		if n2.integer {
			return Number{d: n1.d.Minus(decimal.Int(n2.i))}
		} else {
			return Number{d: n1.d.Minus(n2.d)}
		}
	}
}

func (n1 Number) Module(n2 Number) Number {
	if n1.integer {
		if n2.integer {
			if n2.i == 0 {
				panic("zero division")
			}
			return Number{i: n1.i % n2.i, integer: true}
		} else {
			if n2.d.IsZero() {
				panic("zero division")
			}
			return Number{d: decimal.Int(n1.i).Module(n2.d)}
		}
	} else {
		if n2.integer {
			if n2.i == 0 {
				panic("zero division")
			}
			return Number{d: n1.d.Module(decimal.Int(n2.i))}
		} else {
			if n2.d.IsZero() {
				panic("zero division")
			}
			return Number{d: n1.d.Module(n2.d)}
		}
	}
}

func (n1 Number) Multiplied(n2 Number) Number {
	if n1.integer {
		if n2.integer {
			var e = n1.i * n2.i
			if n1.i != 0 && e/n1.i != n2.i {
				return Number{d: decimal.Int(n1.i).Multiplied(decimal.Int(n2.i))}
			}
			return Number{i: e, integer: true}
		} else {
			return Number{d: decimal.Int(n1.i).Multiplied(n2.d)}
		}
	} else {
		if n2.integer {
			if n2.i == 0 {
				return Number{}
			}
			return Number{d: n1.d.Multiplied(decimal.Int(n2.i))}
		} else {
			return Number{d: n1.d.Multiplied(n2.d)}
		}
	}
}

func (n Number) Opposite() Number {
	if n.integer {
		return Number{i: -n.i, integer: true}
	}
	return Number{d: n.d.Opposite()}
}

func (n1 Number) Plus(n2 Number) Number {
	if n1.integer {
		if n2.integer {
			s := n1.i + n2.i
			if n1.i > 0 && n2.i > 0 && s < 0 || n1.i < 0 && n2.i < 0 && s >= 0 {
				return Number{d: decimal.Int(n1.i).Plus(decimal.Int(n2.i))}
			}
			return Number{i: s, integer: true}
		} else {
			return Number{d: n2.d.Plus(decimal.Int(n1.i))}
		}
	} else {
		if n2.integer {
			return Number{d: n1.d.Plus(decimal.Int(n2.i))}
		} else {
			return Number{d: n1.d.Plus(n2.d)}
		}
	}
}

func (n Number) Rounded(decimals int, mode string) Number {
	if n.integer {
		return n
	}
	return Number{d: n.d.Rounded(decimals, mode)}
}

func (n Number) String() string {
	if n.integer {
		return strconv.Itoa(n.i)
	}
	s := n.d.String()
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		if s[len(s)-1] == '.' {
			s = s[:len(s)-1]
		}
	}
	return s
}
