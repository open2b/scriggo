// Go version: go1.11.5

package lzw

import original "compress/lzw"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"NewReader": original.NewReader,
	"NewWriter": original.NewWriter,
	"Order": reflect.TypeOf(original.Order(int(0))),
}
