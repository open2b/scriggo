// run

package main

import "fmt"

func main() {

	p := fmt.Println

	// TODO: testare anche l'ordine di chiamata

	// Assign to a variable.
	{
		v := 10.01
		p(v)
		v += 20
		p(v)
		v *= 0.3
		p(v)
	}

	// Assign to slice index.
	{
		index := func() int {
			p("called index")
			return 2
		}
		s := []int{1, 2, 3}
		p(s)

		s[0] += 20
		p(s)

		s[index()] += 2
		p(s)

		s[index()] *= index()
		s[index()] /= (index() + 3)
		p(s)
	}

	// Assign to map index.
	{
		key := func() string {
			p("called key")
			return "a key"
		}
		m := map[string]float64{}
		p(m)

		m["x"] += 10.3
		p(m)

		m[key()] *= 2
		p(m)

		m[key()] /= float64(len(key()))
		p(m)
	}

	// Assign to a closure var.
	{
		a := 10
		func() {
			p(a)
			a += 20
			p(a)
			a *= 3
		}()
	}

	// Assign to struct field.
	{
		// https://github.com/open2b/scriggo/issues/478
		// type S struct{ A int }
		// s := func() *S {
		// 	p("called s")
		// 	return &S{}
		// }
		// p(s())
		// s().A = 10
		// s().A += 3
	}

	// IncDec statement on a variable.
	{
		a := 2
		p(a)
		a++
		p(a)
		a--
		a--
		p(a)
	}

	// IncDec statement on a slice indexing.
	{
		s := []int{1, 2, 3, 4}
		p(s)
		s[0]++
		s[1]++
		p(s)
		index := func() int {
			p("index called")
			return 3
		}
		s[index()]++
		p(s)
		s[index()]--
		p(s)
	}
}
