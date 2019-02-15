// Go version: go1.11.5

package tar

import original "archive/tar"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ErrFieldTooLong": &original.ErrFieldTooLong,
	"ErrHeader": &original.ErrHeader,
	"ErrWriteAfterClose": &original.ErrWriteAfterClose,
	"ErrWriteTooLong": &original.ErrWriteTooLong,
	"FileInfoHeader": original.FileInfoHeader,
	"Format": reflect.TypeOf(original.Format(int(0))),
	"Header": reflect.TypeOf(original.Header{}),
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
