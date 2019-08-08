// compile

package main

var v1 = func() chan *[50]int { return nil }()

func f1() chan *[50]int {
	return nil
}

func f2(chan int) {

}

func f3(c chan int) chan []int {
	return nil
}

func main() {}
