//
//  author       : Marco Gazerro <gazerro@open2b.com>
//  initial date : 12/06/2007
//
// date : 09/02/2013
//
// Copyright (c) 2002-2013 Open2b Software Snc. All Rights Reserved.
//
// WARNING: This software is protected by international copyright laws.
// Redistribution in part or in whole strictly prohibited.
//

package template

import (
	"regexp"
	"strconv"
	"strings"
)

var ifReg = regexp.MustCompile(`^\s*(not\s)?\s*(.+?)(?:\(([1-9][0-9]*)\))?\s*$`)
var forReg = regexp.MustCompile(`^(.+)(?:\(([1-9][0-9]*)\))$`)

func Render(document []Node, content map[string]interface{}) string {
	if content == nil {
		content = map[string]interface{}{}
	}
	return render(document, content, content)
}

func render(document []Node, content map[string]interface{}, global map[string]interface{}) string {

	var rendered = ""

	for _, node := range document {
		switch node.Kind {
		case Text:
			rendered += node.Value
		case Show:
			if value := content[node.Value]; value != nil {
				if str, ok := value.(string); ok {
					rendered += str
				}
			} else if strings.HasPrefix(node.Value, "Design.") {
				rendered += global[node.Value].(string)
			}
		case If:
			if len(node.Children) > 0 {
				if m := ifReg.FindStringSubmatch(node.Value); m != nil {
					var isNegated, name, index = m[1], m[2], m[3]
					var value = content[name]
					if value != nil && index != "" {
						var i, _ = strconv.Atoi(index)
						if list, ok := value.([]map[string]interface{}); ok && i > 0 && i <= len(list) {
							value = list[i-1]
						}
					}
					var isTrue = false
					if value != nil {
						switch value.(type) {
						case bool:
							isTrue = value.(bool)
						case string:
							isTrue = len(value.(string)) > 0 && rune(value.(string)[0]) != '0'
						case int:
							isTrue = value.(int) > 0
						//case []map[string]interface{}:
						//    isTrue = len(value.([]map[string]interface{})) > 0
						default:
							panic("unknow value type for '" + name + "'")
						}
					}
					if (isTrue && isNegated == "") || (!isTrue && isNegated != "") {
						rendered += render(node.Children, content, global)
					}
				}
			}
		case For:
			if len(node.Children) > 0 {
				var value interface{}
				if m := forReg.FindStringSubmatch(node.Value); m != nil {
					var name, index = m[1], m[2]
					if value = content[name]; value != nil {
						var i, _ = strconv.Atoi(index)
						if list, ok := value.([]map[string]interface{}); ok && i > 0 && i <= len(list) {
							value = list[i-1]
						} else {
							continue
						}
					}
				} else {
					value = content[node.Value]
				}
				if value != nil {
					if list, ok := value.([]map[string]interface{}); ok {
						for _, element := range list {
							rendered += render(node.Children, element, global)
						}
					} else if element, ok := value.(map[string]interface{}); ok {
						rendered += render(node.Children, element, global)
					}
				}
			}
		case Include, Region:
			if len(node.Children) > 0 {
				if node.Children != nil && len(node.Children) > 0 {
					//rendered += `<div style="border: 2px solid red"><div style="background: red; color: white;">` + node.Value + `</div>`
					rendered += render(node.Children, content, global)
					//rendered += `</div>`
				}
			}
		}
	}

	return rendered
}
