package scrigo

import (
	"testing"
)

func TestScrigoImport(t *testing.T) {

	t.Errorf("unable to run TestScrigoImport")

	// cases := []compiler.MapReader{

	// 	// just "main".
	// 	compiler.MapReader(map[string][]byte{
	// 		"/main.go": []byte(
	// 			`package main
	// 			func main() {
	// 			}`),
	// 	}),

	// 	// "main" importing "pkg".
	// 	compiler.MapReader(map[string][]byte{
	// 		"/main.go": []byte(
	// 			`package main
	// 			import "pkg"
	// 			func main() {
	// 				pkg.F()
	// 			}`),
	// 		"/pkg.go": []byte(
	// 			`package pkg
	// 			func F() {
	// 				println("called pkg.F()")
	// 			}`),
	// 	}),

	// 	// TODO (Gianluca):
	// 	// // "main" importing "pkg1" and "pkg2", where "pkg1" imports "pkg2".
	// 	// parser.MapReader(map[string][]byte{
	// 	// 	"/main.go": []byte(
	// 	// 		`package main
	// 	// 		import "pkg1"
	// 	// 		import "pkg2"
	// 	// 		func main() {
	// 	// 			pkg1.F1()
	// 	// 			pkg2.F2()
	// 	// 		}`),
	// 	// 	"/pkg1.go": []byte(
	// 	// 		`package pkg1
	// 	// 		import "pkg2"
	// 	// 		func F1() {
	// 	// 			pkg2.F2()
	// 	// 		}`),
	// 	// 	"/pkg2.go": []byte(
	// 	// 		`package pkg2
	// 	// 		func F2() {
	// 	// 			println("hi!")
	// 	// 		}`),
	// 	// }),

	// 	// "main" importing "pkg1" and "pkg2".
	// 	compiler.MapReader(map[string][]byte{
	// 		"/main.go": []byte(
	// 			`package main
	// 			import "pkg1"
	// 			import "pkg2"
	// 			func main() {
	// 				pkg1.F1()
	// 				pkg2.F2()
	// 			}`),
	// 		"/pkg1.go": []byte(
	// 			`package pkg1
	// 			func F1() {
	// 				println("called pkg1.F1()")
	// 			}`),
	// 		"/pkg2.go": []byte(
	// 			`package pkg2
	// 			func F2() {
	// 				println("called pkg2.F2()")
	// 			}`),
	// 	}),

	// 	// "main" importing "pkg1" importing "pkg2" (1).
	// 	compiler.MapReader(map[string][]byte{
	// 		"/main.go": []byte(
	// 			`package main
	// 			import "pkg1"
	// 			func main() {
	// 				pkg1.F()
	// 			}`),
	// 		"/pkg1.go": []byte(
	// 			`package pkg1
	// 			import "pkg2"
	// 			func F() {
	// 				pkg2.G()
	// 			}`),
	// 		"/pkg2.go": []byte(
	// 			`package pkg2
	// 			func G() {
	// 				println("called pkg2.G()")
	// 			}`),
	// 	}),

	// 	// "main" importing "pkg1" importing "pkg2" (2).
	// 	compiler.MapReader(map[string][]byte{
	// 		"/main.go": []byte(
	// 			`package main
	// 			import . "pkg1"
	// 			func main() {
	// 				F()
	// 			}`),
	// 		"/pkg1.go": []byte(
	// 			`package pkg1
	// 			import p2 "pkg2"
	// 			func F() {
	// 				p2.G()
	// 			}`),
	// 		"/pkg2.go": []byte(
	// 			`package pkg2
	// 			func G() {
	// 				println("called pkg1.G()")
	// 			}`),
	// 	}),
	// }
	// for i, r := range cases {
	// 	program, err := Compile("/main.go", r, nil)
	// 	if err != nil {
	// 		t.Errorf("test %d/%d, compiling error: %s", i+1, len(cases), err)
	// 		continue
	// 	}
	// 	err = Execute(program)
	// 	if err != nil {
	// 		t.Errorf("test %d/%d, execution error: %s", i+1, len(cases), err)
	// 		continue
	// 	}
	// }

}
