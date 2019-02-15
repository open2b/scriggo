// Go version: go1.11.5

package hex

import original "encoding/hex"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Decode": original.Decode,
	"DecodeString": original.DecodeString,
	"DecodedLen": original.DecodedLen,
	"Dump": original.Dump,
	"Dumper": original.Dumper,
	"Encode": original.Encode,
	"EncodeToString": original.EncodeToString,
	"EncodedLen": original.EncodedLen,
	"ErrLength": &original.ErrLength,
	"InvalidByteError": reflect.TypeOf(original.InvalidByteError(byte(0))),
	"NewDecoder": original.NewDecoder,
	"NewEncoder": original.NewEncoder,
}
