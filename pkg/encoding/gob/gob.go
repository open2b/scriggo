// Go version: go1.11.5

package gob

import original "encoding/gob"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CommonType": reflect.TypeOf(original.CommonType{}),
	"Decoder": reflect.TypeOf(original.Decoder{}),
	"Encoder": reflect.TypeOf(original.Encoder{}),
	"GobDecoder": reflect.TypeOf((*original.GobDecoder)(nil)).Elem(),
	"GobEncoder": reflect.TypeOf((*original.GobEncoder)(nil)).Elem(),
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
	"Register": original.Register,
	"RegisterName": original.RegisterName,
}
