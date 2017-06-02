package inject

import (
	"errors"
	"fmt"
	"reflect"
)

type provider struct {
	constructor interface{}
	argPtrs     []interface{}
}

// NewProvider specifies how to construct a value given its constructor function and argument pointers
func NewProvider(constructor interface{}, argPtrs ...interface{}) Provider {
	fnValue := reflect.ValueOf(constructor)
	if fnValue.Kind() != reflect.Func {
		panicSafe(fmt.Errorf("constructor (%v) is not a function, found %v", fnValue, fnValue.Kind()))
	}

	fnType := reflect.TypeOf(constructor)
	switch fnType.NumOut() {
	case 1:
	case 2:
		if fnType.Out(1).String() != "error" {
			panicSafe(fmt.Errorf("constructor second return value must be an error: %s", fnType.Out(1).String()))
		}
	default:
		panicSafe(fmt.Errorf("constructor must have exactly 1 return value, or 1 return value and an error, found %v", fnType.NumOut()))
	}

	argCount := fnType.NumIn()
	if argCount != len(argPtrs) {
		panicSafe(fmt.Errorf("argPtrs (%d) must match constructor arguments (%d)", len(argPtrs), argCount))
	}

	for i, argPtr := range argPtrs {
		if reflect.TypeOf(argPtr).Kind() != reflect.Ptr {
			panicSafe(fmt.Errorf("argPtrs must all be pointers, found %v", reflect.TypeOf(argPtr)))
		}
		if reflect.ValueOf(argPtr).Elem().Kind() != fnType.In(i).Kind() {
			panicSafe(errors.New("argPtrs must match constructor argument types"))
		}
	}

	return provider{
		constructor: constructor,
		argPtrs:     argPtrs,
	}
}

// Provide returns the result of executing the constructor with argument values resolved from a dependency graph
func (p provider) Provide(g Graph) reflect.Value {
	fnType := reflect.TypeOf(p.constructor)

	argCount := fnType.NumIn()
	args := make([]reflect.Value, argCount, argCount)
	for i := 0; i < argCount; i++ {
		arg := g.Resolve(p.argPtrs[i])
		argType := arg.Type()
		inType := fnType.In(i)
		if !argType.AssignableTo(inType) {
			if !argType.ConvertibleTo(inType) {
				panicSafe(fmt.Errorf(
					"arg %d of type %q cannot be assigned or converted to type %q for provider constructor (%s)",
					i, argType, inType, p.constructor,
				))
			}
			arg = arg.Convert(inType)
		}
		args[i] = arg
	}

	results := reflect.ValueOf(p.constructor).Call(args)
	if len(results) > 1 && !results[1].IsNil() {
		err := results[1].Elem().Interface().(error)
		if err != nil {
			panicSafe(fmt.Errorf("error calling provider constructor for provider (%s): \n error: %s", p.String(), err.Error()))
		}
	}

	return results[0]
}

// Type returns the type of value to expect from Provide
func (p provider) ReturnType() reflect.Type {
	return reflect.TypeOf(p.constructor).Out(0)
}

// String returns a multiline string representation of the provider
func (p provider) String() string {
	return fmt.Sprintf("&provider{\n%s,\n%s\n}",
		indent(fmt.Sprintf("constructor: %s", reflect.TypeOf(p.constructor)), 1),
		indent(fmt.Sprintf("argPtrs: %s", p.fmtArgPtrs()), 1),
	)
}

func (p provider) fmtArgPtrs() string {
	b := make([]string, len(p.argPtrs), len(p.argPtrs))
	for i, argPtr := range p.argPtrs {
		b[i] = ptrString(argPtr)
	}
	return arrayString(b)
}
