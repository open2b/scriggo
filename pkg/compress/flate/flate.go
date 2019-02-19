// Go version: go1.11.5

package flate

import "scrigo"
import "reflect"
import original "compress/flate"

var Package = scrigo.Package{
	"BestCompression": scrigo.Constant(original.BestCompression, nil),
	"BestSpeed": scrigo.Constant(original.BestSpeed, nil),
	"CorruptInputError": reflect.TypeOf(original.CorruptInputError(int64(0))),
	"DefaultCompression": scrigo.Constant(original.DefaultCompression, nil),
	"HuffmanOnly": scrigo.Constant(original.HuffmanOnly, nil),
	"InternalError": reflect.TypeOf(""),
	"NewReader": original.NewReader,
	"NewReaderDict": original.NewReaderDict,
	"NewWriter": original.NewWriter,
	"NewWriterDict": original.NewWriterDict,
	"NoCompression": scrigo.Constant(original.NoCompression, nil),
	"ReadError": reflect.TypeOf(original.ReadError{}),
	"Reader": reflect.TypeOf((*original.Reader)(nil)).Elem(),
	"Resetter": reflect.TypeOf((*original.Resetter)(nil)).Elem(),
	"WriteError": reflect.TypeOf(original.WriteError{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
