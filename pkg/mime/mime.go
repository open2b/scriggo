// Go version: go1.11.5

package mime

import original "mime"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AddExtensionType": original.AddExtensionType,
	"ErrInvalidMediaParameter": &original.ErrInvalidMediaParameter,
	"ExtensionsByType": original.ExtensionsByType,
	"FormatMediaType": original.FormatMediaType,
	"ParseMediaType": original.ParseMediaType,
	"TypeByExtension": original.TypeByExtension,
	"WordDecoder": reflect.TypeOf(original.WordDecoder{}),
	"WordEncoder": reflect.TypeOf(original.WordEncoder(byte(0))),
}
