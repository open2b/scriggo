//
//  author       : Marco Gazerro <gazerro@open2b.com>
//  initial date : 19/02/2009
//
// date : 28/01/2013
//
// Copyright (c) 2002-2013 Open2b Software Snc. All Rights Reserved.
//

package template

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

var _ = fmt.Sprint("")

func NewParser(root string, filter func([]byte) ([]byte, bool)) *Parser {
	return &Parser{root, filter, map[string][]Node{}}
}

// Esegue il parsing della pagina di template a root+path e ritorna l'albero dei nodi
func (parser *Parser) Parse(path string) ([]Node, error) {

	path = path[1:]

	// TODO: togliere il carattere BOM e validare Unicode
	source, err := ioutil.ReadFile(parser.root + path)
	if err != nil {
		return nil, err
	}
	if len(source) == 0 {
		return []Node{{Text, "", nil}}, nil
	}

	instance, err := parser.GetInstance(source)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.File = parser.root + path
		}
		return nil, err
	}

	var page []Node

	if instance != nil {
		source = nil
		instance.File = path
		var dwtPath = parser.root + instance.Template
		// TODO: togliere il carattere BOM e validare Unicode
		var dwtSource, err = ioutil.ReadFile(dwtPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, &Error{"Dwt file '" + instance.Template + "' doesn't exist", parser.root + path, instance.Index, instance.Length}
			} else {
				return nil, err
			}
		}
		page, err = parser.parseSource(dwtSource, instance, "", false)
		if err != nil {
			if e, ok := err.(*Error); ok && e.File == "" {
				e.File = dwtPath
			}
			return nil, err
		}
	} else {
		page, err = parser.parseSource(source, nil, "", false)
		if err != nil {
			if e, ok := err.(*Error); ok && e.File == "" {
				e.File = parser.root + path
			}
			return nil, err
		}
	}

	return page, nil
}

var instanceBeginReg = regexp.MustCompile(`<!--\s*InstanceBegin(\s[^>]*?)?-->`)
var instanceEndReg = regexp.MustCompile(`<!--\s*InstanceEnd(?:\s[^>]*?)-->`)
var htmlTagReg = regexp.MustCompile(`<html[\s>]`)
var templateParameterReg = regexp.MustCompile(`\stemplate="(.*?)"`)
var isLockedReg = regexp.MustCompile(`\scodeOutsideHTMLIsLocked="(.*?)"`)
var docTypeReg = regexp.MustCompile(`^\s*(?:<!DOCTYPE .*?>)?\s*(?:<html\s[\s\S]*?>)?\s*$`)
var editableReg = regexp.MustCompile(`<!--\s*InstanceBeginEditable\s+name="([\s\S]*?)"\s*--(>[\s\S]*?)<!--\s*InstanceEndEditable\s*-->`)

func (parser *Parser) GetInstance(source []byte) (*Instance, error) {

	var loc = instanceBeginReg.FindSubmatchIndex(source)
	if loc == nil {
		return nil, nil
	}

	var index = loc[0]
	var length = loc[1] - loc[0]
	var previous = source[:index]
	var parameters = source[loc[1*2]:loc[1*2+1]]
	var parametersIndex = loc[1*2]

	if htmlTagReg.Match(previous) {
		return nil, nil
	}

	//
	// template

	var template string
	var templateIndex, templateLength int
	{
		var loc = templateParameterReg.FindIndex(parameters)
		if loc == nil {
			return nil, &Error{"Missing attribute 'template'", "", index, length}
		}
		template = string(parameters[loc[0]+11 : loc[1]-1])

		templateIndex = parametersIndex + loc[0] + 1
		templateLength = loc[1] - loc[0] - 1

		if template == "" {
			return nil, &Error{"Missing Dwt file name", "", templateIndex, templateLength}
		}
		if !strings.HasSuffix(template, ".dwt") {
			return nil, &Error{"Dwt file extension must be '.dwt'", "", templateIndex, templateLength}
		}
		if strings.Contains(template, "..") {
			return nil, &Error{"Dwt file name can't contain '..'", "", templateIndex, templateLength}
		}
		if strings.ContainsRune(template, '\\') {
			return nil, &Error{"Dwt file name can't contain '\\'", "", templateIndex, templateLength}
		}
	}

	//
	// codeOutsideHTMLIsLocked

	{
		var loc = isLockedReg.FindIndex(parameters)
		if loc == nil {
			return nil, &Error{"Missing attribute 'codeOutsideHTMLIsLocked'", "", index, length}
		}

		var islocked = string(parameters[loc[0]+26 : loc[1]-1])

		if islocked != "" && islocked != "true" && islocked != "false" {
			var index = parametersIndex + loc[0] + 1
			var length = loc[1] - loc[0] - 1
			return nil, &Error{"codeOutsideHTMLIsLocked can only be empty, 'true' or 'false'", "", index, length}
		}
	}

	if !docTypeReg.Match(previous) {
		return nil, &Error{"InstanceBegin can only follow tags '<DOCTYPE>' and '<html>'", "", index, length}
	}

	if !instanceEndReg.Match(source) {
		return nil, &Error{"Can't find 'InstanceEnd'", "", index, length}
	}

	//
	// regions

	var regions = map[string]RegionBlock{}
	for _, loc := range editableReg.FindAllSubmatchIndex(source, -1) {
		if loc[1*2] == -1 {
			return nil, &Error{"Region name is empty", "", loc[0], loc[1]}
		}
		var name = string(source[loc[1*2]:loc[1*2+1]])
		if _, ok := regions[name]; ok {
			return nil, &Error{"Region '" + name + "' is already defined", "", loc[0], loc[2*2] + 1 - loc[0]}
		}
		regions[name] = RegionBlock{index, source[loc[2*2]+1 : loc[2*2+1]]}
	}

	return &Instance{"", templateIndex, templateLength, template, regions}, nil
}

//
// Private Methods
//

func (parser *Parser) parseSource(source []byte, instance *Instance, currentPath string, skipFilter bool) ([]Node, error) {

	if currentPath == "" {
		currentPath = "/"
	}

	var reader = NewReader(source)

	var parents = []*Node{{None, "", nil}}

	for reader.Read() {

		var kind = reader.Kind

		if kind == End {
			parents = parents[1:]
			continue
		}

		var err error
		var nodes []Node
		var parent = parents[0]
		var value = reader.Value()

		// se il padre è Show o Region ...
		if parent.Kind == Show || parent.Kind == Region {
			// ... se non è testo...
			if kind != Text {
				// ... torna un errore
				return nil, &Error{"Tags are not allowed in 'show' and 'region'", "", reader.Index, reader.Length}
			}
			// ...altrimenti lo scarta
			continue
		}

		// se il padre è 'translate' ...
		if parent.Kind == Translate {
			if kind != Text {
				return nil, &Error{"Tags are non allowed between '{' and '}'", "", reader.Index, reader.Length}
			}
			// ... lo prende come nome dell'espressione da tradurre
			var name = html.UnescapeString(string(value))
			if strings.ContainsAny(name, "=~") {
				return nil, &Error{"Characters '//', '=' and '~' are not allowed between '{' and '}'", "", reader.Index, reader.Length}
			}
			name = strings.TrimSpace(name)
			//$name =~ s/\s+/ /g;   TODO!!!!
			parent.Value = strings.ToLower(name)
			continue
		}

		switch kind {

		case Text:
			if parser.filter != nil && !skipFilter {
				var filteredValue, reParse = parser.filter(value)
				if reParse {
					nodes, err = parser.parseSource(filteredValue, instance, currentPath, true)
					if err != nil {
						return nil, err
					}
				}
			}
			if len(nodes) == 0 {
				nodes = []Node{{Text, string(value), nil}}
			}

		case Include:
			var fileName = path.Clean(string(value))
			var filePath = fileName
			if rune(filePath[0]) != '/' {
				filePath = currentPath + fileName
			}

			if _, ok := parser.includes[filePath]; !ok {
				var fullPath = path.Join(parser.root, filePath[1:])
				source, err = ioutil.ReadFile(fullPath)
				if err != nil {
					if os.IsNotExist(err) {
						return nil, &Error{"Include file doesn't exist", "", reader.Index, reader.Length}
					} else {
						return nil, &Error{"Can't read included file", "", reader.Index, reader.Length}
					}
				}
				var children []Node
				if len(source) > 0 {
					children, err = parser.parseSource(source, nil, path.Dir(filePath)+"/", false)
					if err != nil {
						return nil, err
					}
				}
				parser.includes[filePath] = children
			}

			if _, ok := parser.includes[filePath]; ok {
				var pathName = filePath
				//$pathName    =~ s/\\/\//g;
				nodes = []Node{{Include, pathName, parser.includes[filePath]}}
			}

		case Region:
			if instance == nil {
				return nil, &Error{"Regions are not allowed in this context", "", reader.Index, reader.Length}
			}

			if region, ok := instance.Regions[string(value)]; ok {
				children, err := parser.parseSource(region.content, nil, "", false)
				if err != nil {
					if e, ok := err.(Error); ok {
						e.File = instance.File
						e.Index += region.index
					}
					return nil, err
				}
				nodes = []Node{{Region, "", children}}
			}

			// unshift parents
			var newParents = make([]*Node, len(parents)+1)
			newParents[0] = &Node{Region, string(value), nil}
			copy(newParents[1:], parents)
			parents = newParents

		case Translate:
			nodes = []Node{{kind, "", nil}}
			// unshift @parents, $[]Node[0]
			var newParents = make([]*Node, len(parents)+1)
			newParents[0] = &(nodes[0])
			copy(newParents[1:], parents)
			parents = newParents

		default:
			// if, for or show
			nodes = []Node{{kind, string(bytes.ToLower(value)), nil}}
			// unshift @parents, $[]Node[0]
			var newParents = make([]*Node, len(parents)+1)
			newParents[0] = &(nodes[0])
			copy(newParents[1:], parents)
			parents = newParents

		}

		// aggiunge i nodi come figli del nodo padre
		parent.Children = append(parent.Children, nodes...)

	}

	if reader.Error != nil {
		return nil, reader.Error
	}

	return parents[0].Children, nil
}

type Parser struct {
	root     string
	filter   func([]byte) ([]byte, bool)
	includes map[string][]Node
}

type Node struct {
	Kind     Kind
	Value    string
	Children []Node
}

type RegionBlock struct {
	index   int
	content []byte
}

type Instance struct {
	File     string
	Index    int
	Length   int
	Template string
	Regions  map[string]RegionBlock
}
