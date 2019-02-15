// Go version: go1.11.5

package expvar

import original "expvar"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Do": original.Do,
	"Float": reflect.TypeOf(original.Float{}),
	"Func": reflect.TypeOf((original.Func)(nil)),
	"Get": original.Get,
	"Handler": original.Handler,
	"Int": reflect.TypeOf(original.Int{}),
	"KeyValue": reflect.TypeOf(original.KeyValue{}),
	"Map": reflect.TypeOf(original.Map{}),
	"NewFloat": original.NewFloat,
	"NewInt": original.NewInt,
	"NewMap": original.NewMap,
	"NewString": original.NewString,
	"Publish": original.Publish,
	"String": reflect.TypeOf(original.String{}),
	"Var": reflect.TypeOf((*original.Var)(nil)).Elem(),
}
