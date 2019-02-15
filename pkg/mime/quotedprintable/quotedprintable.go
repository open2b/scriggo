// Go version: go1.11.5

package quotedprintable

import original "mime/quotedprintable"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Reader": reflect.TypeOf(original.Reader{}),
	"Writer": reflect.TypeOf(original.Writer{}),
}
