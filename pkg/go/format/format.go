// Go version: go1.11.5

package format

import original "go/format"
import "scrigo"

var Package = scrigo.Package{
	"Node": original.Node,
	"Source": original.Source,
}
