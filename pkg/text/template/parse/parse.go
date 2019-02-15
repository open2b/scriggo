// Go version: go1.11.5

package parse

import "scrigo"
import "reflect"
import original "text/template/parse"

var Package = scrigo.Package{
	"ActionNode": reflect.TypeOf(original.ActionNode{}),
	"BoolNode": reflect.TypeOf(original.BoolNode{}),
	"BranchNode": reflect.TypeOf(original.BranchNode{}),
	"ChainNode": reflect.TypeOf(original.ChainNode{}),
	"CommandNode": reflect.TypeOf(original.CommandNode{}),
	"DotNode": reflect.TypeOf(original.DotNode{}),
	"FieldNode": reflect.TypeOf(original.FieldNode{}),
	"IdentifierNode": reflect.TypeOf(original.IdentifierNode{}),
	"IfNode": reflect.TypeOf(original.IfNode{}),
	"IsEmptyTree": original.IsEmptyTree,
	"ListNode": reflect.TypeOf(original.ListNode{}),
	"New": original.New,
	"NewIdentifier": original.NewIdentifier,
	"NilNode": reflect.TypeOf(original.NilNode{}),
	"Node": reflect.TypeOf((*original.Node)(nil)).Elem(),
	"NodeType": reflect.TypeOf(original.NodeType(int(0))),
	"NumberNode": reflect.TypeOf(original.NumberNode{}),
	"Parse": original.Parse,
	"PipeNode": reflect.TypeOf(original.PipeNode{}),
	"Pos": reflect.TypeOf(original.Pos(int(0))),
	"RangeNode": reflect.TypeOf(original.RangeNode{}),
	"StringNode": reflect.TypeOf(original.StringNode{}),
	"TemplateNode": reflect.TypeOf(original.TemplateNode{}),
	"TextNode": reflect.TypeOf(original.TextNode{}),
	"Tree": reflect.TypeOf(original.Tree{}),
	"VariableNode": reflect.TypeOf(original.VariableNode{}),
	"WithNode": reflect.TypeOf(original.WithNode{}),
}
