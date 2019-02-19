// Go version: go1.11.5

package mime

import original "mime"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"AddExtensionType": original.AddExtensionType,
	"BEncoding": scrigo.Constant(original.BEncoding, nil),
	"ErrInvalidMediaParameter": &original.ErrInvalidMediaParameter,
	"ExtensionsByType": original.ExtensionsByType,
	"FormatMediaType": original.FormatMediaType,
	"ParseMediaType": original.ParseMediaType,
	"QEncoding": scrigo.Constant(original.QEncoding, nil),
	"TypeByExtension": original.TypeByExtension,
	"WordDecoder": reflect.TypeOf(original.WordDecoder{}),
	"WordEncoder": reflect.TypeOf(original.WordEncoder(byte(0))),
}
