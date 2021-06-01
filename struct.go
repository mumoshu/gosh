package gosh

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Part of the code derived from https://github.com/AdamSLevy/flagbind
//
// Copyright (c) 2020 Adam S Levy
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

type Filter func(name string) string

var defaultFilter = func(s string) string {
	return s
}

// structFieldsReflector is used to map the fields of a struct into flags of a flag.FlagSet
type structFieldsReflector struct {
	TagToEnvName    Filter
	TagToUsage      Filter
	FieldToFlagName Filter
}

func (f *structFieldsReflector) SetStruct(cmd string, v reflect.Value, args []interface{}) error {
	if v.Type().Kind() != reflect.Ptr || v.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("invalid type of value: %v type=%T kind=%s canset=%v", v, v, v.Type().Kind(), v.CanSet())
	}

	if len(args) == 0 {
		return nil
	}

	if len(args) == 1 {
		sv := reflect.ValueOf(args[0])

		if sv.Type().AssignableTo(v.Elem().Type()) {
			v.Elem().Set(sv)

			return nil
		}
	}

	var flags []string

	for i, a := range args {
		s, ok := a.(string)
		if !ok {
			return fmt.Errorf("arg at %d in %v must be string, but was %T", i, args, a)
		}

		flags = append(flags, s)
	}

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)

	if err := f.walkFields(fs, "", v.Elem(), v.Type().Elem()); err != nil {
		return fmt.Errorf("walk fields %s: %w", cmd, err)
	}

	if err := fs.Parse(flags); err != nil {
		return fmt.Errorf("parse %s: %w", cmd, err)
	}

	return nil
}

func (f *structFieldsReflector) AddFlags(from interface{}, flagSet *flag.FlagSet) error {
	v := reflect.ValueOf(from)
	t := v.Type()
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return f.walkFields(flagSet, "", v.Elem(), t.Elem())
	}

	return fmt.Errorf("can only fill from struct pointer, but it was %s", t.Kind())
}

func (f *structFieldsReflector) walkFields(flagSet *flag.FlagSet, prefix string,
	structVal reflect.Value, structType reflect.Type) error {

	for i := 0; i < structVal.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structVal.Field(i)

		switch field.Type.Kind() {
		case reflect.Struct:
			err := f.walkFields(flagSet, prefix+field.Name, fieldValue, field.Type)
			if err != nil {
				return fmt.Errorf("failed to process %s of %s: %w", field.Name, structType.String(), err)
			}

		case reflect.Ptr:
			if fieldValue.CanSet() && field.Type.Elem().Kind() == reflect.Struct {
				// fill the pointer with a new struct of their type
				fieldValue.Set(reflect.New(field.Type.Elem()))

				err := f.walkFields(flagSet, field.Name, fieldValue.Elem(), field.Type.Elem())
				if err != nil {
					return fmt.Errorf("failed to process %s of %s: %w", field.Name, structType.String(), err)
				}
			}

		default:
			addr := fieldValue.Addr()
			// make sure it is exported/public
			if addr.CanInterface() {
				err := f.processField(flagSet, addr.Interface(), prefix+field.Name, field.Type, field.Tag)
				if err != nil {
					return fmt.Errorf("failed to process %s of %s: %w", field.Name, structType.String(), err)
				}
			}
		}
	}

	return nil
}

var (
	durationType          = reflect.TypeOf(time.Duration(0))
	stringSliceType       = reflect.TypeOf([]string{})
	stringToStringMapType = reflect.TypeOf(map[string]string{})
)

func (f *structFieldsReflector) processField(flagSet *flag.FlagSet, fieldRef interface{},
	name string, t reflect.Type, tag reflect.StructTag) (err error) {

	var envName string
	if override, exists := tag.Lookup("env"); exists {
		envName = override
	} else if f.TagToEnvName != nil {
		envName = f.TagToEnvName(name)
	}

	usage := f.TagToUsage(tag.Get("usage"))
	if envName != "" {
		usage = fmt.Sprintf("%s (env %s)", usage, envName)
	}

	tagDefault, hasDefaultTag := tag.Lookup("default")

	var renamed string
	if override, exists := tag.Lookup("flag"); exists {
		if override == "" {
			// empty flag override signal to skip this field
			return nil
		}
		renamed = override
	} else {
		renamed = f.FieldToFlagName(name)
	}

	switch {
	case t.Kind() == reflect.String:
		f.processString(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Bool:
		err = f.processBool(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Float64:
		err = f.processFloat64(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	// NOTE check time.Duration before int64 since it is aliased from int64
	case t == durationType:
		err = f.processDuration(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Int64:
		err = f.processInt64(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Int:
		err = f.processInt(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Uint64:
		err = f.processUint64(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t.Kind() == reflect.Uint:
		err = f.processUint(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t == stringSliceType:
		f.processStringSlice(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

	case t == stringToStringMapType:
		f.processStringToStringMap(fieldRef, hasDefaultTag, tagDefault, flagSet, renamed, usage)

		// ignore any other types
	}

	if err != nil {
		return err
	}

	if envName != "" {
		if val, exists := os.LookupEnv(envName); exists {
			err := flagSet.Lookup(renamed).Value.Set(val)
			if err != nil {
				return fmt.Errorf("failed to set from environment variable %s: %w",
					envName, err)
			}
		}
	}

	return nil
}

func (f *structFieldsReflector) processStringToStringMap(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) {
	casted := fieldRef.(*map[string]string)
	var val map[string]string
	if hasDefaultTag {
		val = parseStringToStringMap(tagDefault)
		*casted = val
	} else if *casted == nil {
		val = make(map[string]string)
		*casted = val
	} else {
		val = *casted
	}
	flagSet.Var(&strToStrMapVar{val: val}, renamed, usage)
}

func (f *structFieldsReflector) processStringSlice(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) {
	casted := fieldRef.(*[]string)
	if hasDefaultTag {
		*casted = parseStringSlice(tagDefault)
	}
	flagSet.Var(&strSliceVar{ref: casted}, renamed, usage)
}

func (f *structFieldsReflector) processUint(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*uint)
	var defaultVal uint
	if hasDefaultTag {
		var asInt int
		asInt, err = strconv.Atoi(tagDefault)
		defaultVal = uint(asInt)
		if err != nil {
			return fmt.Errorf("failed to parse default into uint: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.UintVar(casted, renamed, defaultVal, usage)
	return err
}

func (f *structFieldsReflector) processUint64(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*uint64)
	var defaultVal uint64
	if hasDefaultTag {
		defaultVal, err = strconv.ParseUint(tagDefault, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse default into uint64: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.Uint64Var(casted, renamed, defaultVal, usage)
	return err
}

func (f *structFieldsReflector) processInt(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*int)
	var defaultVal int
	if hasDefaultTag {
		defaultVal, err = strconv.Atoi(tagDefault)
		if err != nil {
			return fmt.Errorf("failed to parse default into int: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.IntVar(casted, renamed, defaultVal, usage)
	return err
}

func (f *structFieldsReflector) processInt64(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*int64)
	var defaultVal int64
	if hasDefaultTag {
		defaultVal, err = strconv.ParseInt(tagDefault, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse default into int64: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.Int64Var(casted, renamed, defaultVal, usage)
	return nil
}

func (f *structFieldsReflector) processDuration(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*time.Duration)
	var defaultVal time.Duration
	if hasDefaultTag {
		defaultVal, err = time.ParseDuration(tagDefault)
		if err != nil {
			return fmt.Errorf("failed to parse default into time.Duration: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.DurationVar(casted, renamed, defaultVal, usage)
	return nil
}

func (f *structFieldsReflector) processFloat64(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*float64)
	var defaultVal float64
	if hasDefaultTag {
		defaultVal, err = strconv.ParseFloat(tagDefault, 64)
		if err != nil {
			return fmt.Errorf("failed to parse default into float64: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.Float64Var(casted, renamed, defaultVal, usage)
	return nil
}

func (f *structFieldsReflector) processBool(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) (err error) {
	casted := fieldRef.(*bool)
	var defaultVal bool
	if hasDefaultTag {
		defaultVal, err = strconv.ParseBool(tagDefault)
		if err != nil {
			return fmt.Errorf("failed to parse default into bool: %w", err)
		}
	} else {
		defaultVal = *casted
	}
	flagSet.BoolVar(casted, renamed, defaultVal, usage)
	return nil
}

func (f *structFieldsReflector) processString(fieldRef interface{}, hasDefaultTag bool, tagDefault string, flagSet *flag.FlagSet, renamed string, usage string) {
	casted := fieldRef.(*string)
	var defaultVal string
	if hasDefaultTag {
		defaultVal = tagDefault
	} else {
		defaultVal = *casted
	}
	flagSet.StringVar(casted, renamed, defaultVal, usage)
}

type strSliceVar struct {
	ref *[]string
}

func (s *strSliceVar) String() string {
	if s.ref == nil {
		return ""
	}
	return strings.Join(*s.ref, ",")
}

func (s *strSliceVar) Set(val string) error {
	parts := parseStringSlice(val)
	*s.ref = append(*s.ref, parts...)

	return nil
}

func parseStringSlice(val string) []string {
	return strings.Split(val, ",")
}

type strToStrMapVar struct {
	val map[string]string
}

func (s strToStrMapVar) String() string {
	if s.val == nil {
		return ""
	}

	var sb strings.Builder
	first := true
	for k, v := range s.val {
		if !first {
			sb.WriteString(",")
		} else {
			first = false
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
	}
	return sb.String()
}

func (s strToStrMapVar) Set(val string) error {
	content := parseStringToStringMap(val)
	for k, v := range content {
		s.val[k] = v
	}
	return nil
}

func parseStringToStringMap(val string) map[string]string {
	result := make(map[string]string)

	pairs := strings.Split(val, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		} else {
			result[kv[0]] = ""
		}
	}

	return result
}
