//
//  author       : Marco Gazerro <gazerro@open2b.com>
//  initial date : 13/08/2006
//
// date : 24/04/2013
//
// Copyright (c) 2002-2013 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"regexp"
)

func NewReader(source []byte) *Reader {
	var reader = new(Reader)
	reader.source = source
	return reader
}

// Profondità del nodo
func (this *Reader) Depth() int {
	return len(this.parents)
}

// Testo del nodo o nome del marcatore
func (this *Reader) Value() []byte {
	var value []byte
	if this.Kind == Text {
		value = this.source[this.Index : this.Index+this.Length]
	} else {
		value = this.parameter
	}
	return value
}

// Legge il prossimo nodo
func (this *Reader) Read() bool {

	// si posiziona all'inizio del nuovo testo da leggere
	this.Index = this.Index + this.Length
	this.Kind = None
	this.EndKind = None
	this.parameter = nil

	if this.Index == len(this.source) {
		// non è più presente testo da leggere
		this.Length = 0
		if len(this.parents) > 0 {
			var parent = this.parents[0]
			this.Error = &Error{"Tag is not closed", "", parent.index, parent.length}
		}
		return false
	}

	if this.Index == 0 {
		// esegue il primo match
		this._match()
		if this.Error != nil {
			return false
		}
	}

	if !this.match.success {
		// non è presente nessun tag da leggere, quindi è tutto testo fino alla fine
		// il controllo se ci sono tag non chiusi viene fatto alla prossima lettura
		this.Kind = Text
		this.Length = len(this.source) - this.Index

	} else if this.match.index > this.Index {
		// è presente un tag ma c'è del testo da leggere prima  di questo
		// il testo sarà letto subito e il tag alla prossima lettura
		this.Kind = Text
		this.Length = this.match.index - this.Index

	} else {
		// è presente un tag e non è presente nessun testo da leggere prima di questo

		this.Length = this.match.length
		this.Kind = this.match.kind
		this.EndKind = this.match.endKind
		this.parameter = this.match.parameter

		if this.Kind != Include {
			if this.Kind == End {
				if len(this.parents) > 0 {
					// toglie l'utimo parent aggiunto alla lista
					var last = len(this.parents) - 1
					var parent = this.parents[last]
					this.parents = this.parents[:last]
					if this.EndKind == None {
						this.EndKind = parent.kind
					} else if this.EndKind != parent.kind {
						this.Error = &Error{
							fmt.Sprintf("End tag doesn't match. Found %v, expected %v", this.EndKind, parent.kind),
							"", this.Index, this.Length}
						return false
					}
				} else {
					this.Error = &Error{"Unmatched end tag", "", this.Index, this.Length}
					return false
				}
			} else {
				// aggiunge il nodo alla fine della lista dei parents
				this.parents = append(this.parents, Parent{this.Kind, this.match.index, this.match.length})
			}
		}

		// legge il prossimo match perché la prossima lettura deve 
		// avere il prossimo match già pronto
		this._match()
		if this.Error != nil {
			return false
		}

	}

	return true
}

func (this *Reader) Close() {
	this.source = nil
	this.Index = 0
	this.Length = 0
	this.parents = nil
	return
}

func (this *Reader) String() string {
	return fmt.Sprintf("%s [%s] (%d, %d)", kindName[this.Kind], kindName[this.EndKind], this.Index, this.Index+this.Length-1)
}

var re = regexp.MustCompile(`({|}|<!--\s*(\.(?:if|for|show|end)|#include|Template(?:Begin|End)Editable)(?:\s+([\s\S]*?)\s*)?-->|<(\/)?([sS][cC][rR][iI][pP][tT]|[sS][tT][yY][lL][eE])(?:\s[^>]*?)?(\/)?>)`)

//                            1 1 1       2                     2        2                                    3                 1 4    5                                                          6

var includeReg = regexp.MustCompile(`^(file|virtual)="(.*?)"$`)
var regionNameReg = regexp.MustCompile(`^name="(.*?)"$`)

func (this *Reader) _match() {

	var source = this.source[this.Index+this.Length:]

	var success bool
	var index int
	var length int
	var kind Kind = None
	var endKind Kind = None
	var parameter []byte

	// scriptOrStyle può essere "", "script" o "style"
	var scriptOrStyle = ""

	var p = 0

MATCH:
	if loc := re.FindSubmatchIndex(source); loc != nil {

		// se è "script" o "style" ...
		if loc[5*2] >= 0 {
			var text = string(source[loc[5*2]:loc[5*2+1]])
			if loc[4*2] >= 0 {
				// chiusura: </script> o </style>
				if scriptOrStyle == "" || text != scriptOrStyle {
					this.Error = &Error{"Start tag does not match", "", p + loc[0], loc[1] - loc[0]}
					return
				}
				scriptOrStyle = ""
			} else if loc[6*2] == -1 {
				// apertura: <script ...> o <style ...>
				if scriptOrStyle != "" {
					this.Error = &Error{"End tag does not match", "", p + loc[0], loc[1] - loc[0]}
					return
				}
				scriptOrStyle = text
			}
			source = source[loc[1]:]
			p += loc[1]
			goto MATCH
		} else if scriptOrStyle != "" {
			source = source[loc[1]:]
			p += loc[1]
			goto MATCH
		}

		success = true
		index = loc[0]
		length = loc[1] - loc[0]

		switch {
		case rune(source[loc[1*2]]) == '{':
			kind = Translate
		case rune(source[loc[1*2]]) == '}':
			kind = End
			endKind = Translate
		case rune(source[loc[2*2]]) == '.':
			switch rune(source[loc[2*2]+1]) {
			case 'i':
				kind = If
				if loc[3*2] >= 0 {
					parameter = source[loc[3*2]:loc[3*2+1]]
				}
			case 'f':
				kind = For
				if loc[3*2] >= 0 {
					parameter = source[loc[3*2]:loc[3*2+1]]
				}
			case 's':
				kind = Show
				if loc[3*2] >= 0 {
					parameter = source[loc[3*2]:loc[3*2+1]]
				}
			case 'e':
				kind = End
				if loc[3*2] >= 0 {
					switch string(source[loc[3*2]:loc[3*2+1]]) {
					case "if":
						endKind = If
					case "for":
						endKind = For
					case "show":
						endKind = Show
					case "":
					default:
						this.Error = &Error{"Unknow end type", "", this.Index, this.Length}
						return
					}
				}
			}
		case rune(source[loc[2*2]]) == '#':
			kind = Include
			if m := includeReg.FindSubmatch(source[loc[3*2]:loc[3*2+1]]); m != nil {
				var file = m[2]
				if bytes.IndexRune(file, '\\') > 0 {
					this.Error = &Error{"Character '\\' is not allowed in file path", "", index, 0}
					return
				}
				parameter = file
			} else {
				this.Error = &Error{"Missing file name", "", index, 0}
				return
			}
		case rune(source[loc[2*2]]) == 'T':
			if rune(source[loc[2*2]+8]) == 'B' {
				kind = Region
				if m := regionNameReg.FindSubmatch(source[loc[3*2]:loc[3*2+1]]); m != nil {
					parameter = m[1]
				} else {
					this.Error = &Error{"Missing region name", "", 0, 0}
					return
				}
			} else {
				kind = End
				endKind = Region
			}
		default:
			panic("unknow match")
		}

	}

	this.match.success = success
	this.match.index = this.Index + this.Length + p + index
	this.match.length = length
	this.match.kind = kind
	this.match.endKind = endKind
	this.match.parameter = parameter

	return
}

type Kind byte

const (
	None Kind = iota
	Region
	Include
	Text
	Translate
	Show
	If
	For
	End
)

var kindName = []string{"None", "Region", "Include", "Text", "Translate", "Show", "If", "For", "End"}

type Reader struct {
	Index     int
	Length    int
	Kind      Kind
	EndKind   Kind
	Error     *Error
	parents   []Parent
	parameter []byte
	source    []byte
	match     struct {
		success   bool
		kind      Kind
		endKind   Kind
		index     int
		length    int
		parameter []byte
	}
}

type Parent struct {
	kind   Kind
	index  int
	length int
}
