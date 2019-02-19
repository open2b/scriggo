// Go version: go1.11.5

package log

import original "log"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Fatal": original.Fatal,
	"Fatalf": original.Fatalf,
	"Fatalln": original.Fatalln,
	"Flags": original.Flags,
	"LUTC": scrigo.Constant(original.LUTC, nil),
	"Ldate": scrigo.Constant(original.Ldate, nil),
	"Llongfile": scrigo.Constant(original.Llongfile, nil),
	"Lmicroseconds": scrigo.Constant(original.Lmicroseconds, nil),
	"Logger": reflect.TypeOf(original.Logger{}),
	"Lshortfile": scrigo.Constant(original.Lshortfile, nil),
	"LstdFlags": scrigo.Constant(original.LstdFlags, nil),
	"Ltime": scrigo.Constant(original.Ltime, nil),
	"New": original.New,
	"Output": original.Output,
	"Panic": original.Panic,
	"Panicf": original.Panicf,
	"Panicln": original.Panicln,
	"Prefix": original.Prefix,
	"Print": original.Print,
	"Printf": original.Printf,
	"Println": original.Println,
	"SetFlags": original.SetFlags,
	"SetOutput": original.SetOutput,
	"SetPrefix": original.SetPrefix,
}
