// Go version: go1.11.5

package color

import "reflect"
import original "image/color"
import "scrigo"

var Package = scrigo.Package{
	"Alpha": reflect.TypeOf(original.Alpha{}),
	"Alpha16": reflect.TypeOf(original.Alpha16{}),
	"Alpha16Model": &original.Alpha16Model,
	"AlphaModel": &original.AlphaModel,
	"Black": &original.Black,
	"CMYK": reflect.TypeOf(original.CMYK{}),
	"CMYKModel": &original.CMYKModel,
	"CMYKToRGB": original.CMYKToRGB,
	"Color": reflect.TypeOf((*original.Color)(nil)).Elem(),
	"Gray": reflect.TypeOf(original.Gray{}),
	"Gray16": reflect.TypeOf(original.Gray16{}),
	"Gray16Model": &original.Gray16Model,
	"GrayModel": &original.GrayModel,
	"Model": reflect.TypeOf((*original.Model)(nil)).Elem(),
	"ModelFunc": original.ModelFunc,
	"NRGBA": reflect.TypeOf(original.NRGBA{}),
	"NRGBA64": reflect.TypeOf(original.NRGBA64{}),
	"NRGBA64Model": &original.NRGBA64Model,
	"NRGBAModel": &original.NRGBAModel,
	"NYCbCrA": reflect.TypeOf(original.NYCbCrA{}),
	"NYCbCrAModel": &original.NYCbCrAModel,
	"Opaque": &original.Opaque,
	"Palette": reflect.TypeOf((original.Palette)(nil)),
	"RGBA": reflect.TypeOf(original.RGBA{}),
	"RGBA64": reflect.TypeOf(original.RGBA64{}),
	"RGBA64Model": &original.RGBA64Model,
	"RGBAModel": &original.RGBAModel,
	"RGBToCMYK": original.RGBToCMYK,
	"RGBToYCbCr": original.RGBToYCbCr,
	"Transparent": &original.Transparent,
	"White": &original.White,
	"YCbCr": reflect.TypeOf(original.YCbCr{}),
	"YCbCrModel": &original.YCbCrModel,
	"YCbCrToRGB": original.YCbCrToRGB,
}
