// Go version: go1.11.5

package gzip

import "reflect"
import original "compress/gzip"
import "scrigo"

var Package = scrigo.Package{
	"ErrChecksum": &original.ErrChecksum,
	"ErrHeader": &original.ErrHeader,
	"Header": reflect.TypeOf(original.Header{}),
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"NewWriterLevel": original.NewWriterLevel,
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
