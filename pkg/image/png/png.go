// Go version: go1.11.5

package png

import original "image/png"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BestCompression": scrigo.Constant(original.BestCompression, nil),
	"BestSpeed": scrigo.Constant(original.BestSpeed, nil),
	"CompressionLevel": reflect.TypeOf(original.CompressionLevel(int(0))),
	"Decode": original.Decode,
	"DecodeConfig": original.DecodeConfig,
	"DefaultCompression": scrigo.Constant(original.DefaultCompression, nil),
	"Encode": original.Encode,
	"Encoder": reflect.TypeOf(original.Encoder{}),
	"EncoderBuffer": reflect.TypeOf(original.EncoderBuffer{}),
	"EncoderBufferPool": reflect.TypeOf((*original.EncoderBufferPool)(nil)).Elem(),
	"FormatError": reflect.TypeOf(""),
	"NoCompression": scrigo.Constant(original.NoCompression, nil),
	"UnsupportedError": reflect.TypeOf(""),
}
