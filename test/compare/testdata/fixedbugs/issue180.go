// run

package main

import "fmt"

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

}
