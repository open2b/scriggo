// Go version: go1.11.5

package ast

import original "go/ast"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"ArrayType": reflect.TypeOf(original.ArrayType{}),
	"AssignStmt": reflect.TypeOf(original.AssignStmt{}),
	"BadDecl": reflect.TypeOf(original.BadDecl{}),
	"BadExpr": reflect.TypeOf(original.BadExpr{}),
	"BadStmt": reflect.TypeOf(original.BadStmt{}),
	"BasicLit": reflect.TypeOf(original.BasicLit{}),
	"BinaryExpr": reflect.TypeOf(original.BinaryExpr{}),
	"BlockStmt": reflect.TypeOf(original.BlockStmt{}),
	"BranchStmt": reflect.TypeOf(original.BranchStmt{}),
	"CallExpr": reflect.TypeOf(original.CallExpr{}),
	"CaseClause": reflect.TypeOf(original.CaseClause{}),
	"ChanDir": reflect.TypeOf(original.ChanDir(int(0))),
	"ChanType": reflect.TypeOf(original.ChanType{}),
	"CommClause": reflect.TypeOf(original.CommClause{}),
	"Comment": reflect.TypeOf(original.Comment{}),
	"CommentGroup": reflect.TypeOf(original.CommentGroup{}),
	"CommentMap": reflect.TypeOf((original.CommentMap)(nil)),
	"CompositeLit": reflect.TypeOf(original.CompositeLit{}),
	"Decl": reflect.TypeOf((*original.Decl)(nil)).Elem(),
	"DeclStmt": reflect.TypeOf(original.DeclStmt{}),
	"DeferStmt": reflect.TypeOf(original.DeferStmt{}),
	"Ellipsis": reflect.TypeOf(original.Ellipsis{}),
	"EmptyStmt": reflect.TypeOf(original.EmptyStmt{}),
	"Expr": reflect.TypeOf((*original.Expr)(nil)).Elem(),
	"ExprStmt": reflect.TypeOf(original.ExprStmt{}),
	"Field": reflect.TypeOf(original.Field{}),
	"FieldFilter": reflect.TypeOf((original.FieldFilter)(nil)),
	"FieldList": reflect.TypeOf(original.FieldList{}),
	"File": reflect.TypeOf(original.File{}),
	"FileExports": original.FileExports,
	"Filter": reflect.TypeOf((original.Filter)(nil)),
	"FilterDecl": original.FilterDecl,
	"FilterFile": original.FilterFile,
	"FilterPackage": original.FilterPackage,
	"ForStmt": reflect.TypeOf(original.ForStmt{}),
	"Fprint": original.Fprint,
	"FuncDecl": reflect.TypeOf(original.FuncDecl{}),
	"FuncLit": reflect.TypeOf(original.FuncLit{}),
	"FuncType": reflect.TypeOf(original.FuncType{}),
	"GenDecl": reflect.TypeOf(original.GenDecl{}),
	"GoStmt": reflect.TypeOf(original.GoStmt{}),
	"Ident": reflect.TypeOf(original.Ident{}),
	"IfStmt": reflect.TypeOf(original.IfStmt{}),
	"ImportSpec": reflect.TypeOf(original.ImportSpec{}),
	"Importer": reflect.TypeOf((original.Importer)(nil)),
	"IncDecStmt": reflect.TypeOf(original.IncDecStmt{}),
	"IndexExpr": reflect.TypeOf(original.IndexExpr{}),
	"Inspect": original.Inspect,
	"InterfaceType": reflect.TypeOf(original.InterfaceType{}),
	"IsExported": original.IsExported,
	"KeyValueExpr": reflect.TypeOf(original.KeyValueExpr{}),
	"LabeledStmt": reflect.TypeOf(original.LabeledStmt{}),
	"MapType": reflect.TypeOf(original.MapType{}),
	"MergeMode": reflect.TypeOf(original.MergeMode(uint(0))),
	"MergePackageFiles": original.MergePackageFiles,
	"NewCommentMap": original.NewCommentMap,
	"NewIdent": original.NewIdent,
	"NewObj": original.NewObj,
	"NewPackage": original.NewPackage,
	"NewScope": original.NewScope,
	"Node": reflect.TypeOf((*original.Node)(nil)).Elem(),
	"NotNilFilter": original.NotNilFilter,
	"ObjKind": reflect.TypeOf(original.ObjKind(int(0))),
	"Object": reflect.TypeOf(original.Object{}),
	"Package": reflect.TypeOf(original.Package{}),
	"PackageExports": original.PackageExports,
	"ParenExpr": reflect.TypeOf(original.ParenExpr{}),
	"Print": original.Print,
	"RangeStmt": reflect.TypeOf(original.RangeStmt{}),
	"ReturnStmt": reflect.TypeOf(original.ReturnStmt{}),
	"Scope": reflect.TypeOf(original.Scope{}),
	"SelectStmt": reflect.TypeOf(original.SelectStmt{}),
	"SelectorExpr": reflect.TypeOf(original.SelectorExpr{}),
	"SendStmt": reflect.TypeOf(original.SendStmt{}),
	"SliceExpr": reflect.TypeOf(original.SliceExpr{}),
	"SortImports": original.SortImports,
	"Spec": reflect.TypeOf((*original.Spec)(nil)).Elem(),
	"StarExpr": reflect.TypeOf(original.StarExpr{}),
	"Stmt": reflect.TypeOf((*original.Stmt)(nil)).Elem(),
	"StructType": reflect.TypeOf(original.StructType{}),
	"SwitchStmt": reflect.TypeOf(original.SwitchStmt{}),
	"TypeAssertExpr": reflect.TypeOf(original.TypeAssertExpr{}),
	"TypeSpec": reflect.TypeOf(original.TypeSpec{}),
	"TypeSwitchStmt": reflect.TypeOf(original.TypeSwitchStmt{}),
	"UnaryExpr": reflect.TypeOf(original.UnaryExpr{}),
	"ValueSpec": reflect.TypeOf(original.ValueSpec{}),
	"Visitor": reflect.TypeOf((*original.Visitor)(nil)).Elem(),
	"Walk": original.Walk,
}
