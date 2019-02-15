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
	"RuneCount": original.RuneCount,
	"RuneCountInString": original.RuneCountInString,
	"RuneLen": original.RuneLen,
	"RuneStart": original.RuneStart,
	"Valid": original.Valid,
	"ValidRune": original.ValidRune,
	"ValidString": original.ValidString,
}
