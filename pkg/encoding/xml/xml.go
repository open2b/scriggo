// Go version: go1.11.5

package xml

import original "encoding/xml"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Attr": reflect.TypeOf(original.Attr{}),
	"CharData": reflect.TypeOf((original.CharData)(nil)),
	"Comment": reflect.TypeOf((original.Comment)(nil)),
	"CopyToken": original.CopyToken,
	"Decoder": reflect.TypeOf(original.Decoder{}),
	"Directive": reflect.TypeOf((original.Directive)(nil)),
	"Encoder": reflect.TypeOf(original.Encoder{}),
	"EndElement": reflect.TypeOf(original.EndElement{}),
	"Escape": original.Escape,
	"EscapeText": original.EscapeText,
	"HTMLAutoClose": &original.HTMLAutoClose,
	"HTMLEntity": &original.HTMLEntity,
	"Marshal": original.Marshal,
	"MarshalIndent": original.MarshalIndent,
	"Marshaler": reflect.TypeOf((*original.Marshaler)(nil)).Elem(),
	"MarshalerAttr": reflect.TypeOf((*original.MarshalerAttr)(nil)).Elem(),
	"Name": reflect.TypeOf(original.Name{}),
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
	"NewTokenDecoder": original.NewTokenDecoder,
	"ProcInst": reflect.TypeOf(original.ProcInst{}),
	"StartElement": reflect.TypeOf(original.StartElement{}),
	"SyntaxError": reflect.TypeOf(original.SyntaxError{}),
	"TagPathError": reflect.TypeOf(original.TagPathError{}),
	"Token": reflect.TypeOf((*original.Token)(nil)).Elem(),
	"TokenReader": reflect.TypeOf((*original.TokenReader)(nil)).Elem(),
	"Unmarshal": original.Unmarshal,
	"UnmarshalError": reflect.TypeOf(""),
	"Unmarshaler": reflect.TypeOf((*original.Unmarshaler)(nil)).Elem(),
	"UnmarshalerAttr": reflect.TypeOf((*original.UnmarshalerAttr)(nil)).Elem(),
	"UnsupportedTypeError": reflect.TypeOf(original.UnsupportedTypeError{}),
}
