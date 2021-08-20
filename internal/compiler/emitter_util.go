// Copyright (c) 2019 Open2b Software Snc. All rights reserved.
// https://www.open2b.com

// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compiler

import (
	"fmt"
	"reflect"
	"unicode"
	"unicode/utf8"

	"github.com/open2b/scriggo/ast"
	"github.com/open2b/scriggo/internal/runtime"
)

// changeRegister emits the code that move the content of register src to
// register dst, making a conversion if necessary.
func (em *emitter) changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {
	em._changeRegister(k, src, dst, srcType, dstType, false)
}

// changeRegisterConvertFormat behaves like changeRegister but handles a format
// conversion from a value with type 'markdown' to 'html'.
func (em *emitter) changeRegisterConvertFormat(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type) {
	em._changeRegister(k, src, dst, srcType, dstType, true)
}

// _changeRegister should be called only by 'changeRegister' and
// 'changeRegisterMDToHTML'.
func (em *emitter) _changeRegister(k bool, src, dst int8, srcType reflect.Type, dstType reflect.Type, mdToHTML bool) {

	// dst is indirect, so the value must be "typed" to its true (original) type
	// before putting it into general.
	//
	// As an exception to this rule, if srcType is a Scriggo type then the
	// Typify instruction must use the Scriggo internal type or the Go defined
	// type, not the Scriggo defined type; that's because when the Scriggo
	// defined type reaches the outside, the gc compiled code cannot access to
	// it's internal implementation. Think about a struct defined in the gc
	// compiled code: when this is passed through an interface to the gc
	// compiled code, it can make a type assertion with the concrete type
	// because it's defined in the gc compiled code, then it is able to access
	// the struct fields. This is not possible with Scriggo defined types,
	// because the gc compiled code cannot reference to them.
	if dst < 0 {
		if st, ok := srcType.(runtime.ScriggoType); ok {
			srcType = st.GoType()
		}
		em.fb.emitTypify(k, srcType, src, dst)
		return
	}

	srcKind := srcType.Kind()
	dstKind := dstType.Kind()

	if dstKind == reflect.Interface && srcKind == reflect.Interface {
		em.fb.emitMove(k, src, dst, srcKind)
		return
	}

	// When moving a value from int, float or string to general, value's type
	// must be "typed" to its true (original) type.
	if dstKind == reflect.Interface {
		em.fb.emitTypify(k, srcType, src, dst)
		return
	}

	// Source register is different than destination register: a conversion is
	// needed.
	if dstKind != srcKind || mdToHTML {
		if k {
			em.fb.enterScope()
			tmp := em.fb.newRegister(srcKind)
			em.fb.emitMove(true, src, tmp, srcKind)
			em.fb.emitConvert(tmp, dstType, dst, srcKind)
			em.fb.exitScope()
		}
		em.fb.emitConvert(src, dstType, dst, srcKind)
		return
	}

	if k || src != dst {
		em.fb.emitMove(k, src, dst, srcKind)
	}

}

// comparisonWithZeroInteger returns the non-constant integer expression if
// cond is a binary operator expression with one of the following forms:
//
//    expr == 0
//    0    == expr
//    expr != 0
//    0    != expr
//
// If the previous condition does not apply, nil is returned.
//
func (em *emitter) comparisonWithZeroInteger(cond *ast.BinaryOperator) ast.Expression {

	// The operator must be a comparison between a constant and a non-constant
	// expression.
	var expr, constant ast.Expression
	if ti2 := em.ti(cond.Expr2); ti2 != nil && ti2.IsConstant() {
		constant = cond.Expr2
		expr = cond.Expr1
	} else if ti1 := em.ti(cond.Expr1); ti1 != nil && ti1.IsConstant() {
		constant = cond.Expr1
		expr = cond.Expr2
	}

	// The expression can't be nil.
	if expr == nil {
		return nil
	}

	// The expression must have an associated type info.
	exprTi := em.ti(expr)
	if exprTi == nil {
		return nil
	}

	// The expression type can't be nil.
	if exprTi.Type == nil {
		return nil
	}

	// The expression kind must be an integer.
	if k := exprTi.Type.Kind(); k < reflect.Int || k > reflect.Uintptr {
		return nil
	}

	// The constant must be 0.
	if ti := em.ti(constant); ti != nil && ti.HasValue() {
		if i64, ok := ti.value.(int64); ok && i64 != 0 {
			return nil
		}
	}

	return expr
}

// compositeLiteralLen returns the length of a composite literal.
func (em *emitter) compositeLiteralLen(node *ast.CompositeLiteral) int {
	size := 0
	for _, kv := range node.KeyValues {
		if kv.Key != nil {
			key := int(em.ti(kv.Key).Constant.int64())
			if key > size {
				size = key
			}
		}
		size++
	}
	return size
}

// stackDifference returns the difference of registers between a and b.
func stackDifference(a, b runtime.StackShift) runtime.StackShift {
	return runtime.StackShift{
		a[0] - b[0],
		a[1] - b[1],
		a[2] - b[2],
		a[3] - b[3],
	}
}

// isDefault reports whether the given case clause is the "default" case.
func isDefault(clause *ast.Case) bool {
	return clause.Expressions == nil
}

// isExported reports whether name is exported, according to
// https://golang.org/ref/spec#Exported_identifiers.
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.Is(unicode.Lu, r)
}

// isPredeclNil reports whether expr is the predeclared nil.
func (em *emitter) isPredeclNil(expr ast.Expression) bool {
	return em.ti(expr).Nil()
}

// builtinCallName returns the name of the builtin function in a call
// expression. If expr is not a call expression or the function is not a
// builtin, it returns an empty string.
func (em *emitter) builtinCallName(expr ast.Expression) string {
	if call, ok := expr.(*ast.Call); ok && em.ti(call.Func).IsBuiltinFunction() {
		return call.Func.(*ast.Identifier).Name
	}
	return ""
}

// numOut reports the number of return parameters of call, if it is a function
// call. If is not, returns 0 and false.
func (em *emitter) numOut(call *ast.Call) (int, bool) {
	if ti := em.ti(call.Func); ti != nil && ti.Type != nil {
		if ti.Type.Kind() == reflect.Func {
			return ti.Type.NumOut(), true
		}
	}
	return 0, false
}

// isSpecialCall reports whether f(<args>) is the special call 'f(g(..))'.
func (em *emitter) isSpecialCall(args []ast.Expression) bool {
	if len(args) != 1 {
		return false
	}
	g, gIsCall := args[0].(*ast.Call)
	if !gIsCall {
		return false
	}
	_, ok := em.numOut(g)
	return ok
}

// kindToType returns the internal register type of a reflect kind.
func kindToType(k reflect.Kind) registerType {
	switch k {
	case reflect.Bool:
		return intRegister
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intRegister
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return intRegister
	case reflect.Float32, reflect.Float64:
		return floatRegister
	case reflect.String:
		return stringRegister
	default:
		return generalRegister
	}
}

// newGlobal returns a new Global value. If typ is a Scriggo type, then typ is
// converted to a gc compiled type before creating the Global value.
func newGlobal(pkg, name string, typ reflect.Type, value reflect.Value) Global {
	// TODO: is this solution ok? Or does it prevent from creating "global"
	// values with scriggo types?
	if st, ok := typ.(runtime.ScriggoType); ok {
		typ = st.GoType()
	}
	return Global{
		Pkg:   pkg,
		Name:  name,
		Type:  typ,
		Value: value,
	}
}

// canEmitDirectly reports whether a value of kind k1 can be emitted directly
// into a register of kind k2 without the needing of passing from an
// intermediate register.
//
// The result of this function depends from the current implementation of the
// VM.
func canEmitDirectly(k1, k2 reflect.Kind) bool {
	// If the destination register has an interface kind, it's not possible to
	// emit the value directly: a typify may be necessary.
	if k2 == reflect.Interface {
		return false
	}
	// Functions are handled as special cases in VM.
	if k1 == reflect.Func || k2 == reflect.Func {
		return false
	}
	return kindToType(k1) == kindToType(k2)
}

// setFunctionVarRefs sets the var refs of a function.
// This method operates on the current function builder, so shall be called
// before changing or saving it.
func (em *emitter) setFunctionVarRefs(fn *runtime.Function, closureVars []ast.Upvar) {
	refs := make([]int16, len(closureVars))
	for i := range closureVars {
		v := &closureVars[i]
		// v is predefined.
		if v.Declaration == nil {
			refs[i] = em.varStore.predefVarIndex(v.NativeValue, v.NativeValueType, v.NativePkg, v.NativeName)
			em.varStore.setPredefVarRef(fn, v.NativeValue, int16(i))
			continue
		}
		em.varStore.setClosureVar(fn, v.Declaration.(*ast.Identifier).Name, int16(i))
		// v is a variable declared in function.
		ident := v.Declaration.(*ast.Identifier)
		if em.fb.declaredInFunc(ident.Name) {
			refs[i] = int16(em.fb.scopeLookup(ident.Name))
			continue
		}
		// v is package-level variable. This happens when a function
		// literal assigned to a package-level variable refers to another
		// package-level variable.
		if index, ok := em.varStore.nonLocalVarIndex(ident); ok {
			refs[i] = int16(index)
			continue
		}
		panic(fmt.Sprintf("BUG: don't know how to handle identifier %s", ident))
	}
	fn.VarRefs = refs
}

func (em *emitter) emitValueNotPredefined(ti *typeInfo, reg int8, dstType reflect.Type) (int8, bool) {
	typ := ti.Type
	if reg == 0 {
		return reg, false
	}
	// Handle nil values.
	if ti.value == nil {
		c := em.fb.makeGeneralValue(reflect.ValueOf(nil))
		em.changeRegister(true, c, reg, typ, dstType)
		return reg, false
	}
	switch v := ti.value.(type) {
	case int64:
		c := em.fb.makeIntValue(v)
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			em.fb.emitLoad(c, reg, reflect.Int)
			em.changeRegister(false, reg, reg, typ, dstType)
			return reg, false
		}
		tmp := em.fb.newRegister(typ.Kind())
		em.fb.emitLoad(c, tmp, reflect.Int)
		em.changeRegister(false, tmp, reg, typ, dstType)
		return reg, false
	case float64:
		var c int
		if typ.Kind() == reflect.Float32 {
			c = em.fb.makeFloatValue(float64(float32(v)))
		} else {
			c = em.fb.makeFloatValue(v)
		}
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			em.fb.emitLoad(c, reg, reflect.Float64)
			em.changeRegister(false, reg, reg, typ, dstType)
			return reg, false
		}
		tmp := em.fb.newRegister(typ.Kind())
		em.fb.emitLoad(c, tmp, reflect.Float64)
		em.changeRegister(false, tmp, reg, typ, dstType)
		return reg, false
	case string:
		c := em.fb.makeStringValue(v)
		em.changeRegister(true, c, reg, typ, dstType)
		return reg, false
	}
	v := reflect.ValueOf(ti.value)
	switch v.Kind() {
	case reflect.Interface:
		panic("BUG: not implemented") // remove.
	case reflect.Array:
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			em.fb.emitMakeArray(typ, reg)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(typ.Kind())
		em.fb.emitMakeArray(typ, tmp)
		em.changeRegister(false, tmp, reg, typ, dstType)
		em.fb.exitStack()
	case reflect.Struct:
		if canEmitDirectly(typ.Kind(), dstType.Kind()) {
			em.fb.emitMakeStruct(typ, reg)
			return reg, false
		}
		em.fb.enterStack()
		tmp := em.fb.newRegister(typ.Kind())
		em.fb.emitMakeStruct(typ, tmp)
		em.changeRegister(false, tmp, reg, typ, dstType)
		em.fb.exitStack()
	case reflect.Slice,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Chan,
		reflect.Func,
		reflect.Map,
		reflect.Ptr:
		c := em.fb.makeGeneralValue(v)
		em.changeRegister(true, c, reg, typ, dstType)
	case reflect.UnsafePointer:
		panic("BUG: not implemented") // remove.
	default:
		panic(fmt.Errorf("unsupported value type %T", ti.value))
	}
	return reg, false
}

// emitComparison emits the comparison expression x op y as a sequence of
// instructions where the last one is an 'if' instruction. ky indicates if y
// is a constant.
func (em *emitter) emitComparison(op ast.OperatorType, ky bool, x, y int8, tx, ty reflect.Type, pos *ast.Position) {
	xKind := tx.Kind()
	yKind := ty.Kind()
	var condition runtime.Condition
	switch op {
	case ast.OperatorEqual:
		condition = runtime.ConditionEqual
		if xKind == reflect.Interface || yKind == reflect.Interface {
			condition = runtime.ConditionInterfaceEqual
		}
	case ast.OperatorNotEqual:
		condition = runtime.ConditionNotEqual
		if xKind == reflect.Interface || yKind == reflect.Interface {
			condition = runtime.ConditionInterfaceNotEqual
		}
	default:
		k := tx.Kind()
		if reflect.Uint <= k && k <= reflect.Uintptr {
			switch op {
			case ast.OperatorLess:
				condition = runtime.ConditionLessU
			case ast.OperatorLessEqual:
				condition = runtime.ConditionLessEqualU
			case ast.OperatorGreater:
				condition = runtime.ConditionGreaterU
			case ast.OperatorGreaterEqual:
				condition = runtime.ConditionGreaterEqualU
			default:
				panic("unexpected operator")
			}
		} else {
			switch op {
			case ast.OperatorLess:
				condition = runtime.ConditionLess
			case ast.OperatorLessEqual:
				condition = runtime.ConditionLessEqual
			case ast.OperatorGreater:
				condition = runtime.ConditionGreater
			case ast.OperatorGreaterEqual:
				condition = runtime.ConditionGreaterEqual
			default:
				panic("unexpected operator")
			}
		}
	}
	// If an operand has interface type and the other operand not,
	// change the register of the other operand.
	changeReg := (xKind == reflect.Interface) != (yKind == reflect.Interface)
	if changeReg {
		em.fb.enterStack()
		g := em.fb.newRegister(reflect.Interface)
		if xKind == reflect.Interface {
			em.changeRegister(ky, y, g, ty, tx)
			y = g
			ky = false
		} else {
			em.changeRegister(false, x, g, tx, ty)
			x = g
			xKind = reflect.Interface
		}
	}
	em.fb.emitIf(ky, x, condition, y, xKind, pos)
	if changeReg {
		em.fb.exitStack()
	}
}

// emitContains emits a contains expression 'x contains y' as a sequence of
// instructions where the last one is an 'if' instruction. If 'not' is true,
// it emits 'x not contains y'. ky indicates if y is a constant.
//
// ty is nil if the expression is 'x contains nil' or 'x not contains nil'.
func (em *emitter) emitContains(not, ky bool, x, y int8, tx, ty reflect.Type, pos *ast.Position) {
	var condition runtime.Condition
	var t reflect.Type
	switch tx.Kind() {
	case reflect.String:
		condition = runtime.ConditionContainsRune
		if ty.Kind() == reflect.String {
			condition = runtime.ConditionContainsSubstring
		}
		t = ty
	case reflect.Slice, reflect.Array:
		condition = runtime.ConditionContainsElement
		t = tx.Elem()
	case reflect.Map:
		condition = runtime.ConditionContainsKey
		t = tx.Key()
	default:
		panic("unexpected type")
	}
	if ty == nil {
		condition = runtime.ConditionContainsNil
	}
	if not {
		condition += runtime.ConditionNotContainsSubstring - runtime.ConditionContainsSubstring
	}
	changeReg := t.Kind() == reflect.Interface && ty != nil && ty.Kind() != reflect.Interface
	if changeReg {
		em.fb.enterStack()
		g := em.fb.newRegister(reflect.Interface)
		em.changeRegister(ky, y, g, ty, t)
		y = g
		ky = false
	}
	em.fb.emitIf(ky, x, condition, y, t.Kind(), pos)
	if changeReg {
		em.fb.exitStack()
	}
}

// convertPosition converts an AST position to a runtime position.
func convertPosition(pos *ast.Position) *runtime.Position {
	if pos == nil {
		return nil
	}
	return &runtime.Position{
		Line:   pos.Line,
		Column: pos.Column,
		Start:  pos.Start,
		End:    pos.End,
	}
}
