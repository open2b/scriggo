// Go version: go1.11.5

package image

import original "image"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Alpha": reflect.TypeOf(original.Alpha{}),
	"Alpha16": reflect.TypeOf(original.Alpha16{}),
	"Black": &original.Black,
	"CMYK": reflect.TypeOf(original.CMYK{}),
	"Config": reflect.TypeOf(original.Config{}),
	"Decode": original.Decode,
	"DecodeConfig": original.DecodeConfig,
	"ErrFormat": &original.ErrFormat,
	"Gray": reflect.TypeOf(original.Gray{}),
	"Gray16": reflect.TypeOf(original.Gray16{}),
	"Image": reflect.TypeOf((*original.Image)(nil)).Elem(),
	"NRGBA": reflect.TypeOf(original.NRGBA{}),
	"NRGBA64": reflect.TypeOf(original.NRGBA64{}),
	"NYCbCrA": reflect.TypeOf(original.NYCbCrA{}),
	"NewAlpha": original.NewAlpha,
	"NewAlpha16": original.NewAlpha16,
	"NewCMYK": original.NewCMYK,
	"NewGray": original.NewGray,
	"NewGray16": original.NewGray16,
	"NewNRGBA": original.NewNRGBA,
	"NewNRGBA64": original.NewNRGBA64,
	"NewNYCbCrA": original.NewNYCbCrA,
	"NewPaletted": original.NewPaletted,
	"NewRGBA": original.NewRGBA,
	"NewRGBA64": original.NewRGBA64,
	"NewUniform": original.NewUniform,
	"NewYCbCr": original.NewYCbCr,
	"Opaque": &original.Opaque,
	"Paletted": reflect.TypeOf(original.Paletted{}),
	"PalettedImage": reflect.TypeOf((*original.PalettedImage)(nil)).Elem(),
	"Point": reflect.TypeOf(original.Point{}),
	"Pt": original.Pt,
	"RGBA": reflect.TypeOf(original.RGBA{}),
	"RGBA64": reflect.TypeOf(original.RGBA64{}),
	"Rect": original.Rect,
	"Rectangle": reflect.TypeOf(original.Rectangle{}),
	"RegisterFormat": original.RegisterFormat,
	"Transparent": &original.Transparent,
	"Uniform": reflect.TypeOf(original.Uniform{}),
	"White": &original.White,
	"YCbCr": reflect.TypeOf(original.YCbCr{}),
	"YCbCrSubsampleRatio": reflect.TypeOf(original.YCbCrSubsampleRatio(int(0))),
	"ZP": &original.ZP,
	"ZR": &original.ZR,
}
