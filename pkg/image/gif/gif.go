// Go version: go1.11.5

package gif

import "reflect"
import original "image/gif"
import "scrigo"

var Package = scrigo.Package{
	"Decode": original.Decode,
	"DecodeAll": original.DecodeAll,
	"DecodeConfig": original.DecodeConfig,
	"Encode": original.Encode,
	"EncodeAll": original.EncodeAll,
	"GIF": reflect.TypeOf(original.GIF{}),
	"Options": reflect.TypeOf(original.Options{}),
}
