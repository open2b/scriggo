// Go version: go1.11.5

package filepath

import original "path/filepath"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Abs": original.Abs,
	"Base": original.Base,
	"Clean": original.Clean,
	"Dir": original.Dir,
	"ErrBadPattern": &original.ErrBadPattern,
	"EvalSymlinks": original.EvalSymlinks,
	"Ext": original.Ext,
	"FromSlash": original.FromSlash,
	"Glob": original.Glob,
	"HasPrefix": original.HasPrefix,
	"IsAbs": original.IsAbs,
	"Join": original.Join,
	"Match": original.Match,
	"Rel": original.Rel,
	"SkipDir": &original.SkipDir,
	"Split": original.Split,
	"SplitList": original.SplitList,
	"ToSlash": original.ToSlash,
	"VolumeName": original.VolumeName,
	"Walk": original.Walk,
	"WalkFunc": reflect.TypeOf((original.WalkFunc)(nil)),
}
