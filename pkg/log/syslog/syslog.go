// Go version: go1.11.5

package syslog

import "scrigo"
import "reflect"
import original "log/syslog"

var Package = scrigo.Package{
	"Dial": original.Dial,
	"New": original.New,
	"NewLogger": original.NewLogger,
	"Priority": reflect.TypeOf(original.Priority(int(0))),
	"Writer": reflect.TypeOf(original.Writer{}),
}
