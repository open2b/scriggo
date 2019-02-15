// Go version: go1.11.5

package png

import original "image/png"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"CompressionLevel": reflect.TypeOf(original.CompressionLevel(int(0))),
	"Decode": original.Decode,
	"DecodeConfig": original.DecodeConfig,
	"Encode": original.Encode,
	"Encoder": reflect.TypeOf(original.Encoder{}),
	"EncoderBuffer": reflect.TypeOf(original.EncoderBuffer{}),
	"EncoderBufferPool": reflect.TypeOf((*original.EncoderBufferPool)(nil)).Elem(),
	"FormatError": reflect.TypeOf(""),
	"UnsupportedError": reflect.TypeOf(""),
}
