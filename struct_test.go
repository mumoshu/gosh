package gosh

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStruct(t *testing.T) {
	f := &structFieldsReflector{
		TagToEnvName:    defaultFilter,
		TagToUsage:      defaultFilter,
		FieldToFlagName: defaultFilter,
	}

	type Opts struct {
		Foo string            `flag:"foo"`
		Bar string            `flag:"bar"`
		Num int               `flag:"num"`
		M   map[string]string `flag:"set"`

		// Note that you can't set private fields like this,
		// due to that go's reflection doesn't allow setting a value for a private field.
		foo string
	}

	t.Run("flags", func(t *testing.T) {
		var opts Opts

		err := f.SetStruct("teststruct", reflect.ValueOf(&opts), []interface{}{"-foo=FOO", "-bar", "BAR", "-set=a=A", "-set", "b=B"})

		assert.NoError(t, err)

		assert.Equal(t, Opts{Foo: "FOO", Bar: "BAR", M: map[string]string{"a": "A", "b": "B"}}, opts)
	})

	t.Run("direct", func(t *testing.T) {
		var opts Opts

		err := f.SetStruct("teststruct", reflect.ValueOf(&opts), []interface{}{Opts{Foo: "FOO2", Bar: "BAR2", M: map[string]string{"a": "A", "b": "B"}}})

		assert.NoError(t, err)

		assert.Equal(t, Opts{Foo: "FOO2", Bar: "BAR2", M: map[string]string{"a": "A", "b": "B"}}, opts)
	})
}
