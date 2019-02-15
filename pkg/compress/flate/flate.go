// Go version: go1.11.5

package flate

import "reflect"
import original "compress/flate"
import "scrigo"

var Package = scrigo.Package{
	"CorruptInputError": reflect.TypeOf(original.CorruptInputError(int64(0))),
	"InternalError": reflect.TypeOf(""),
	"NewReader": original.NewReader,
	"NewReaderDict": original.NewReaderDict,
	"NewWriter": original.NewWriter,
	"NewWriterDict": original.NewWriterDict,
	"ReadError": reflect.TypeOf(original.ReadError{}),
	"Reader": reflect.TypeOf((*original.Reader)(nil)).Elem(),
	"Resetter": reflect.TypeOf((*original.Resetter)(nil)).Elem(),
	"WriteError": reflect.TypeOf(original.WriteError{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
