// Go version: go1.11.5

package zlib

import "scrigo"
import "reflect"
import original "compress/zlib"

var Package = scrigo.Package{
	"BestCompression": scrigo.Constant(original.BestCompression, nil),
	"BestSpeed": scrigo.Constant(original.BestSpeed, nil),
	"DefaultCompression": scrigo.Constant(original.DefaultCompression, nil),
	"ErrChecksum": &original.ErrChecksum,
	"ErrDictionary": &original.ErrDictionary,
	"ErrHeader": &original.ErrHeader,
	"HuffmanOnly": scrigo.Constant(original.HuffmanOnly, nil),
	"NewReader": original.NewReader,
	"NewReaderDict": original.NewReaderDict,
	"NewWriter": original.NewWriter,
	"NewWriterLevel": original.NewWriterLevel,
	"NewWriterLevelDict": original.NewWriterLevelDict,
	"NoCompression": scrigo.Constant(original.NoCompression, nil),
	"Resetter": reflect.TypeOf((*original.Resetter)(nil)).Elem(),
	"Writer": reflect.TypeOf(original.Writer{}),
}
