// Go version: go1.11.5

package context

import "scrigo"
import "reflect"
import original "context"

var Package = scrigo.Package{
	"Background": original.Background,
	"CancelFunc": reflect.TypeOf((original.CancelFunc)(nil)),
	"Canceled": &original.Canceled,
	"Context": reflect.TypeOf((*original.Context)(nil)).Elem(),
	"DeadlineExceeded": &original.DeadlineExceeded,
	"TODO": original.TODO,
	"WithCancel": original.WithCancel,
	"WithDeadline": original.WithDeadline,
	"WithTimeout": original.WithTimeout,
	"WithValue": original.WithValue,
}
