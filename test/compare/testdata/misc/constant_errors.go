// errorcheck

package main

func f() int { return 0 }

func main() {

	var ch chan int
	_ = ch

	const _ = len(([]string)(nil)[0])             // ERROR `const initializer len([]string(nil)[0]) is not a constant`
	const _ = len([1]int{<-ch})                   // ERROR `const initializer len([1]int literal) is not a constant`
	const _ = len([1]int{1 * (2 - <-ch)})         // ERROR `const initializer len([1]int literal) is not a constant`
	const _ = len(new([1]int))                    // ERROR `const initializer len(new([1]int)) is not a constant`
	const _ = len(<-make(chan [1]int, 1))         // ERROR `const initializer len(<-make(chan [1]int, 1)) is not a constant`
	const _ = len([3]int{f()})                    // ERROR `const initializer len([3]int literal) is not a constant`
	const _ = len([1][]int{append([]int{}, 0)})   // ERROR `const initializer len([1][]int literal) is not a constant`
	const _ = len([1]int{copy([]int{}, []int{})}) // ERROR `const initializer len([1]int literal) is not a constant`
	const _ = len([1]interface{}{recover()})      // ERROR `const initializer len([1]interface{} literal) is not a constant`

	const _ = cap([1]int{<-ch})                   // ERROR `const initializer cap([1]int literal) is not a constant`
	const _ = cap([1]int{1 * (2 - <-ch)})         // ERROR `const initializer cap([1]int literal) is not a constant`
	const _ = cap(new([1]int))                    // ERROR `const initializer cap(new([1]int)) is not a constant`
	const _ = cap(<-make(chan [1]int, 1))         // ERROR `const initializer cap(<-make(chan [1]int, 1)) is not a constant`
	const _ = cap([3]int{f()})                    // ERROR `const initializer cap([3]int literal) is not a constant`
	const _ = cap([1][]int{append([]int{}, 0)})   // ERROR `const initializer cap([1][]int literal) is not a constant`
	const _ = cap([1]int{copy([]int{}, []int{})}) // ERROR `const initializer cap([1]int literal) is not a constant`
	const _ = cap([1]interface{}{recover()})      // ERROR `const initializer cap([1]interface{} literal) is not a constant`

}
