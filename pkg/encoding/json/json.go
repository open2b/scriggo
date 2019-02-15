// Go version: go1.11.5

package json

import original "encoding/json"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Compact": original.Compact,
	"Decoder": reflect.TypeOf(original.Decoder{}),
	"Delim": reflect.TypeOf(original.Delim(rune(0))),
	"Encoder": reflect.TypeOf(original.Encoder{}),
	"HTMLEscape": original.HTMLEscape,
	"Indent": original.Indent,
	"InvalidUTF8Error": reflect.TypeOf(original.InvalidUTF8Error{}),
	"InvalidUnmarshalError": reflect.TypeOf(original.InvalidUnmarshalError{}),
	"Marshal": original.Marshal,
	"MarshalIndent": original.MarshalIndent,
	"Marshaler": reflect.TypeOf((*original.Marshaler)(nil)).Elem(),
	"MarshalerError": reflect.TypeOf(original.MarshalerError{}),
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
	"Number": reflect.TypeOf(""),
	"RawMessage": reflect.TypeOf((original.RawMessage)(nil)),
	"SyntaxError": reflect.TypeOf(original.SyntaxError{}),
	"Token": reflect.TypeOf((*original.Token)(nil)).Elem(),
	"Unmarshal": original.Unmarshal,
	"UnmarshalFieldError": reflect.TypeOf(original.UnmarshalFieldError{}),
	"UnmarshalTypeError": reflect.TypeOf(original.UnmarshalTypeError{}),
	"Unmarshaler": reflect.TypeOf((*original.Unmarshaler)(nil)).Elem(),
	"UnsupportedTypeError": reflect.TypeOf(original.UnsupportedTypeError{}),
	"UnsupportedValueError": reflect.TypeOf(original.UnsupportedValueError{}),
	"Valid": original.Valid,
}
