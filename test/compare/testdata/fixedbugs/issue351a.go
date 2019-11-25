// run

package main

func main() {
	{
		ss := [][]int{
			{1, 2, 3},
		}
		_ = ss
	}
	{
		data := []struct{ M map[string]int }{
			{M: map[string]int{"k1": 0}},
		}
		_ = data
	}
}
