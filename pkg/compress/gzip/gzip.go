// Go version: go1.11.5

package gzip

import original "compress/gzip"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"BestCompression": scrigo.Constant(original.BestCompression, nil),
	"BestSpeed": scrigo.Constant(original.BestSpeed, nil),
	"DefaultCompression": scrigo.Constant(original.DefaultCompression, nil),
	"ErrChecksum": &original.ErrChecksum,
	"ErrHeader": &original.ErrHeader,
	"Header": reflect.TypeOf(original.Header{}),
	"HuffmanOnly": scrigo.Constant(original.HuffmanOnly, nil),
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"NewWriterLevel": original.NewWriterLevel,
	"NoCompression": scrigo.Constant(original.NoCompression, nil),
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
