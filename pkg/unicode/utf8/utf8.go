// Go version: go1.11.5

package utf8

import original "unicode/utf8"
import "scrigo"

var Package = scrigo.Package{
	"DecodeLastRune": original.DecodeLastRune,
	"DecodeLastRuneInString": original.DecodeLastRuneInString,
	"DecodeRune": original.DecodeRune,
	"DecodeRuneInString": original.DecodeRuneInString,
	"EncodeRune": original.EncodeRune,
	"FullRune": original.FullRune,
	"FullRuneInString": original.FullRuneInString,
	"MaxRune": scrigo.Constant(original.MaxRune, nil),
	"RuneCount": original.RuneCount,
	"RuneCountInString": original.RuneCountInString,
	"RuneError": scrigo.Constant(original.RuneError, nil),
	"RuneLen": original.RuneLen,
	"RuneSelf": scrigo.Constant(original.RuneSelf, nil),
	"RuneStart": original.RuneStart,
	"UTFMax": scrigo.Constant(original.UTFMax, nil),
	"Valid": original.Valid,
	"ValidRune": original.ValidRune,
	"ValidString": original.ValidString,
}
