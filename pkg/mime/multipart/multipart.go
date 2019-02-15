// Go version: go1.11.5

package multipart

import "reflect"
import original "mime/multipart"
import "scrigo"

var Package = scrigo.Package{
	"ErrMessageTooLarge": &original.ErrMessageTooLarge,
	"File": reflect.TypeOf((*original.File)(nil)).Elem(),
	"FileHeader": reflect.TypeOf(original.FileHeader{}),
	"Form": reflect.TypeOf(original.Form{}),
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Part": reflect.TypeOf(original.Part{}),
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
