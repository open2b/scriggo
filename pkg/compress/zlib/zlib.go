// Go version: go1.11.5

package zlib

import original "compress/zlib"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ErrChecksum": &original.ErrChecksum,
	"ErrDictionary": &original.ErrDictionary,
	"ErrHeader": &original.ErrHeader,
	"NewReader": original.NewReader,
	"NewReaderDict": original.NewReaderDict,
	"NewWriter": original.NewWriter,
	"NewWriterLevel": original.NewWriterLevel,
	"NewWriterLevelDict": original.NewWriterLevelDict,
	"Resetter": reflect.TypeOf((*original.Resetter)(nil)).Elem(),
	"Writer": reflect.TypeOf(original.Writer{}),
}
