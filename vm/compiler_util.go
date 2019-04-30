package vm

import "reflect"

// kindToType returns VM's type of k.
func kindToType(k reflect.Kind) Type {
	switch k {
	case reflect.Int,
		reflect.Bool,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Int8,
		reflect.Uint,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uint8,
		reflect.Uintptr:
		return TypeInt
	case reflect.Float64, reflect.Float32:
		return TypeFloat
	case reflect.Invalid:
		panic("unexpected")
	case reflect.String:
		return TypeString
	case reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice,
		reflect.Struct:
		return TypeIface
	case reflect.UnsafePointer:
		panic("TODO: not implemented")
	default:
		panic("bug")
	}
}
