// run

package main

type Int int

type A1 [3]Int
type A2 [56]A1

func main() {
	{
		var a A1
		e := a[0]
		_ = e
	}
	{
		var a A2
		_ = a
	}
	{
		var a [5]A1
		_ = a
	}
	{
		var a [20][]A2
		_ = a
	}
	{
		var a [2]Int
		_ = a
	}
	{
		var a [2]map[Int]int
		e := a[0][0]
		_ = e
	}
	{
		var a [43][][][][][]int
		_ = a
	}
	{
		var a [43][][][][][]Int
		_ = a
		a = [43][][][][][]Int{}
	}
	{
		var a [4][10][32][20]Int
		_ = a
	}
	{
		var a [4][10][32][20]A1
		_ = a
	}
	{
		var a [4][10][32][20]A2
		_ = a
	}
}
