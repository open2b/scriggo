// Go version: go1.11.5

package utf16

import original "unicode/utf16"
import "scrigo"

var Package = scrigo.Package{
	"Decode": original.Decode,
	"DecodeRune": original.DecodeRune,
	"Encode": original.Encode,
	"EncodeRune": original.EncodeRune,
	"IsSurrogate": original.IsSurrogate,
}
