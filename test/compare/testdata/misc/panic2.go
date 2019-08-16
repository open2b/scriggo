// paniccheck

package main

func main() {
	defer func() {
		defer func() {
			func() {
				defer func() {
					v := recover()
					println("recovered", v.(int))
				}()
				defer func() {
					panic(4)
				}()
				panic(3)
			}()
			recover()
			panic(5)
		}()
		panic(2)
	}()
	panic(1)
}
