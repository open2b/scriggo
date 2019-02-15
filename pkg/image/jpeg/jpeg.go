// Go version: go1.11.5

package jpeg

import "reflect"
import original "image/jpeg"
import "scrigo"

var Package = scrigo.Package{
	"Decode": original.Decode,
	"DecodeConfig": original.DecodeConfig,
	"Encode": original.Encode,
	"FormatError": reflect.TypeOf(""),
	"Options": reflect.TypeOf(original.Options{}),
	"Reader": reflect.TypeOf((*original.Reader)(nil)).Elem(),
	"UnsupportedError": reflect.TypeOf(""),
}
