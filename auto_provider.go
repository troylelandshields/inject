package inject

import (
	"errors"
	"fmt"
	"reflect"
)

type autoProvider struct {
	constructor interface{}
}

// NewAutoProvider specifies how to construct a value given its constructor function.
// Argument values are auto-resolved by type.
func NewAutoProvider(constructor interface{}) Provider {
	fnValue := reflect.ValueOf(constructor)
	if fnValue.Kind() != reflect.Func {
		panicSafe(errors.New("constructor is not a function"))
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

	return autoProvider{
		constructor: constructor,
	}
}

// Provide returns the result of executing the constructor with argument values resolved by type from a dependency graph
func (p autoProvider) Provide(g Graph) reflect.Value {
	fnType := reflect.TypeOf(p.constructor)

	argCount := fnType.NumIn()
	args := make([]reflect.Value, argCount, argCount)
	for i := 0; i < argCount; i++ {
		argType := fnType.In(i)
		values := g.ResolveByType(argType)
		if len(values) == 0 {
			values = g.ResolveByAssignableType(argType)
		}

		if len(values) > 1 {
			panicSafe(fmt.Errorf("more than one defined pointer is assignable to the provider argument %d of type (%v)", i, argType))
		} else if len(values) == 0 {
			panicSafe(fmt.Errorf("no defined pointer is assignable to the provider argument %d of type (%v)", i, argType))
		}
		args[i] = values[0]
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
func (p autoProvider) ReturnType() reflect.Type {
	return reflect.TypeOf(p.constructor).Out(0)
}

// String returns a multiline string representation of the autoProvider
func (p autoProvider) String() string {
	return fmt.Sprintf("&autoProvider{\n%s\n}",
		indent(fmt.Sprintf("constructor: %s", reflect.TypeOf(p.constructor)), 1),
	)
}
