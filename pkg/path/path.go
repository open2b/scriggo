// Go version: go1.11.5

package path

import original "path"
import "scrigo"

var Package = scrigo.Package{
	"Base": original.Base,
	"Clean": original.Clean,
	"Dir": original.Dir,
	"ErrBadPattern": &original.ErrBadPattern,
	"Ext": original.Ext,
	"IsAbs": original.IsAbs,
	"Join": original.Join,
	"Match": original.Match,
	"Split": original.Split,
}
