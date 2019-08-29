// run

package main

import (
	"archive/tar"
	"fmt"
)

func main() {

	// Variable
	{
		a := 10
		fmt.Println(a)
		a = 20
		fmt.Println(a)
	}

	// Pointer indirection
	{
		{
			a := 2
			pa := &a

			fmt.Println(a)

			*pa = 70

			fmt.Println(a)
		}
		{
			a := 3
			s := []*int{&a}
			fmt.Println(a)
			*(s[0]) = -20
			fmt.Println(a)
		}
	}

	// Slice indexing operation
	{
		{
			s := []int{1, 2, 3}
			fmt.Println(s)
			s[2] = -4
			fmt.Println(s)
		}
		{
			ss := [][]int{
				{1, 2, 3},
				{4, 5, 6},
				{7, 8, 9},
			}
			fmt.Println(ss)
			ss[1][2] = -432
			fmt.Println(ss)
		}
		{
			sss := [][][]string{
				[][]string{
					// TODO(Gianluca): fails with not-explicit type.
					// {"a", "b", "c"},
					// {"d", "e", "f"},
					[]string{"a", "b", "c"},
					[]string{"d", "e", "f"},
				},
			}
			fmt.Println(sss)
			sss[0][1][2] = "F"
			fmt.Println(sss)
		}
	}

	// Field selector of an addressable struct operand
	{
		type T = struct {
			F int
		}
		s := []T{
			T{5},
			T{6},
		}
		fmt.Println("s: ", s)
		s[0].F = -42
		fmt.Println("s: ", s)
	}

	// Array indexing operation of an addressable array
	{
		{
			a := [3][3][3]int{}
			fmt.Println(a)
			a[1][2][1] = 5
			fmt.Println(a)
		}
	}

	// Map index expression.
	{
		{
			m := map[string][]int{
				"a": []int{97},
				"b": []int{98},
			}
			fmt.Println(m)
			m["b"][0] = 99
			fmt.Println(m)
		}
	}

	// Miscellaneous
	{
		{
			data := []struct{ M map[string]int }{
				struct{ M map[string]int }{M: map[string]int{"k1": 0}},
			}
			fmt.Println(data)
			data[0].M["k1"] = 3
			fmt.Println(data)
			data[0].M["k3"] = 10
			fmt.Println(data)
		}
		{
			h := tar.Header{}
			fmt.Print(h.Name)
			h.Name = "test"
			fmt.Print(h.Name)
			hsm := map[string][]tar.Header{"first": []tar.Header{h}}
			fmt.Print(hsm["first"][0].Name)
			hsm["first"][0].Name = "set"
			fmt.Print(hsm["first"][0].Name)
		}
	}

}
