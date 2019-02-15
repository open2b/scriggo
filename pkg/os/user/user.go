// Go version: go1.11.5

package user

import original "os/user"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Current": original.Current,
	"Group": reflect.TypeOf(original.Group{}),
	"Lookup": original.Lookup,
	"LookupGroup": original.LookupGroup,
	"LookupGroupId": original.LookupGroupId,
	"LookupId": original.LookupId,
	"UnknownGroupError": reflect.TypeOf(""),
	"UnknownGroupIdError": reflect.TypeOf(""),
	"UnknownUserError": reflect.TypeOf(""),
	"UnknownUserIdError": reflect.TypeOf(original.UnknownUserIdError(int(0))),
	"User": reflect.TypeOf(original.User{}),
}
