// Go version: go1.11.5

package asn1

import original "encoding/asn1"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BitString": reflect.TypeOf(original.BitString{}),
	"Enumerated": reflect.TypeOf(original.Enumerated(int(0))),
	"Flag": reflect.TypeOf(false),
	"Marshal": original.Marshal,
	"MarshalWithParams": original.MarshalWithParams,
	"NullBytes": &original.NullBytes,
	"NullRawValue": &original.NullRawValue,
	"ObjectIdentifier": reflect.TypeOf((original.ObjectIdentifier)(nil)),
	"RawContent": reflect.TypeOf((original.RawContent)(nil)),
	"RawValue": reflect.TypeOf(original.RawValue{}),
	"StructuralError": reflect.TypeOf(original.StructuralError{}),
	"SyntaxError": reflect.TypeOf(original.SyntaxError{}),
	"Unmarshal": original.Unmarshal,
	"UnmarshalWithParams": original.UnmarshalWithParams,
}
