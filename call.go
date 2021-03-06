package gosh

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/mumoshu/gosh/context"
)

func CallFunc(ctx context.Context, name string, fun interface{}, funArgs ...interface{}) ([]reflect.Value, error) {
	fv := reflect.ValueOf(fun)
	x := reflect.TypeOf(fun)

	args, err := getArgs(ctx, name, x, funArgs)
	if err != nil {
		return nil, err
	}

	// fmt.Fprintf(os.Stderr, "%v\n", args)

	// for o := 0; o < numOut; o++ {
	// 	returnV := x.Out(0)
	// 	return_Kind := returnV.Kind()
	// 	fmt.Printf("\nParameter OUT: "+strconv.Itoa(o)+"\nKind: %v\nName: %v\n", return_Kind, returnV.Name())
	// }

	panicked := true
	defer func() {
		if panicked {
			fmt.Fprintf(context.Stderr(ctx), "Panicked while running %q with %v\n", name, args)
		}
	}()
	values := fv.Call(args)
	panicked = false

	if len(values) > 0 {
		last := values[len(values)-1]

		err, ok := last.Interface().(error)
		if ok {
			return values, err
		}
	}

	return values, nil
}

func CallMethod(ctx context.Context, name string, m reflect.Value, funArgs ...interface{}) ([]reflect.Value, error) {
	args, err := getArgs(ctx, name, m.Type(), funArgs)
	if err != nil {
		return nil, err
	}

	values := m.Call(args)

	if len(values) > 0 {
		last := values[len(values)-1]

		err, ok := last.Interface().(error)
		if ok {
			return values, err
		}
	}

	return values, nil
}

type testingTKey struct{}

func getArgs(ctx context.Context, cmdName string, x reflect.Type, funArgs []interface{}) ([]reflect.Value, error) {
	numIn := x.NumIn()
	// numOut := x.NumOut()

	// funcName := x.String()
	isVariadic := x.IsVariadic()
	// pkgPath := x.PkgPath()

	// fmt.Fprintf(os.Stderr, "gosh.Call: funcName=%s, numIn=%d, isVariadic=%v, pkgPath=%s, funArgs=%v\n", funcName, numIn, isVariadic, pkgPath, funArgs)

	args := make([]reflect.Value, numIn)

	// https://coderwall.com/p/b5dpwq/fun-with-the-reflection-package-to-analyse-any-function
FOR:
	for i, j := 0, 0; i < numIn; i++ {
		inV := x.In(i)
		in_Kind := inV.Kind() //func
		in_typeName := inV.String()

		reflectTypeContext := reflect.TypeOf(ctx)
		reflectTypeTestingT := reflect.TypeOf(&testing.T{})

		// fmt.Fprintf(os.Stderr, "i=%d, type=%v, kind=%v\n", i, inV, in_Kind)

		switch in_Kind {
		case reflect.Ptr:
			if reflectTypeTestingT.AssignableTo(inV) {
				v := ctx.Value(testingTKey{})

				if v == nil {
					panic("Missing *testing.T in context. Probably you tried to export a function that takes *testing.T outside of a go test?")
				}
				args[i] = reflect.ValueOf(v)
			} else {
				return nil, fmt.Errorf("parameter %v at %d is not supported", in_typeName, i)
			}
		case reflect.Interface:
			// if inV != reflectTypeContext {
			// 	panic(fmt.Errorf("param %d is interface but not %v", i, reflectTypeContext))
			// }
			if !reflectTypeContext.AssignableTo(inV) {
				return nil, fmt.Errorf("param %d is interface %v but not assignable from %v", i, in_Kind, reflectTypeContext)
			}
			args[i] = reflect.ValueOf(ctx)
		case reflect.String:
			if len(funArgs)-1 < j {
				panic(fmt.Errorf("missing argument for required parameter %v at %d", in_Kind, j))
			}
			a := funArgs[j]
			j++
			args[i] = reflect.ValueOf(a)
		case reflect.Bool:
			var v interface{}
			var err error
			switch a := funArgs[j].(type) {
			case string:
				v, err = strconv.ParseBool(a)
				if err != nil {
					panic(err)
				}
			default:
				v = a
			}
			j++
			args[i] = reflect.ValueOf(v)
		case reflect.Int:
			if len(funArgs)-1 < j {
				panic(fmt.Errorf("missing argument for required parameter %v at %d", in_Kind, j))
			}

			var v interface{}
			switch a := funArgs[j].(type) {
			case string:
				intv, err := strconv.ParseInt(a, 10, 0)
				if err != nil {
					panic(err)
				}
				v = int(intv)
			default:
				v = a
			}
			j++
			args[i] = reflect.ValueOf(v)
		case reflect.Slice:
			if i == numIn-1 && isVariadic {
				args = args[:i]
				for _, v := range funArgs[j:] {
					args = append(args, reflect.ValueOf(v))
				}
				break FOR
			}

			switch inV.Elem().Kind() {
			case reflect.String:
				var strArgs []string
				for _, v := range funArgs[j:] {
					strArgs = append(strArgs, v.(string))
				}
				args[i] = reflect.ValueOf(strArgs)
			default:
				panic(fmt.Errorf("slice of %v is not yet supported", inV.Elem().Kind()))
			}

			break FOR
		case reflect.Map:
			args[i] = reflect.ValueOf(funArgs[j])
		case reflect.Struct:
			f := &structFieldsReflector{
				TagToEnvName:    defaultFilter,
				TagToUsage:      defaultFilter,
				FieldToFlagName: defaultFilter,
			}

			// This returns a pointer to the value of the type i.e. new(foo), &foo{}, instead of foo{}.
			v := reflect.New(inV)

			flagArgs := funArgs[j:]

			if err := f.SetStruct(cmdName, v, funArgs[j:]); err != nil {
				return nil, fmt.Errorf("failed to map args to %v, for args starting at %d, %v: %v", inV.Name(), j, flagArgs, err)
			}

			// And that's why you need to take the Elem, which is the underlying value the pointer points.
			// Otherwise you get errors like `Call using *gosh_test.Opts as type gosh_test.Opts`
			args[i] = v.Elem()

			break FOR
		default:
			panic(fmt.Sprintf("call: unsupported func parameter name=%v type=%v kind=%v while trying to match argument: %v", inV.Name(), in_typeName, in_Kind, funArgs[j]))
		}
	}

	return args, nil
}
