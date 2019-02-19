// Go version: go1.11.5

package draw

import original "image/draw"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Draw": original.Draw,
	"DrawMask": original.DrawMask,
	"Drawer": reflect.TypeOf((*original.Drawer)(nil)).Elem(),
	"FloydSteinberg": &original.FloydSteinberg,
	"Image": reflect.TypeOf((*original.Image)(nil)).Elem(),
	"Op": reflect.TypeOf(original.Op(int(0))),
	"Over": scrigo.Constant(original.Over, nil),
	"Quantizer": reflect.TypeOf((*original.Quantizer)(nil)).Elem(),
	"Src": scrigo.Constant(original.Src, nil),
}
