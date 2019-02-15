// Go version: go1.11.5

package md5

import original "crypto/md5"
import "scrigo"

var Package = scrigo.Package{
	"New": original.New,
	"Sum": original.Sum,
}
