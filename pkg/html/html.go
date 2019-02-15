// Go version: go1.11.5

package html

import original "html"
import "scrigo"

var Package = scrigo.Package{
	"EscapeString": original.EscapeString,
	"UnescapeString": original.UnescapeString,
}
