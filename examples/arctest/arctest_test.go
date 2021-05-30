package arctest_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/examples/arctest"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestArctest(t *testing.T) {
	arctest := arctest.New()

	if err := arctest.Run("all", "tmpdir", false, 1); err != nil {
		t.Fatal(err)
	}
}

func TestBashEnv(t *testing.T) {
	sh := arctest.New()

	goshtest.Run(t, sh, func() {
		fmt.Fprintf(os.Stderr, "%v\n", os.Args)
		var stdout, stderr bytes.Buffer

		err := sh.Run(t, "bash", "-c", "hello world", gosh.WriteStdout(&stdout), gosh.WriteStderr(&stderr))

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "hello world\n", stdout.String())
		assert.Equal(t, "hello world (stderr)\n", stderr.String())
		// assert.Equal(t, "", stderr.String())
	})
}

func TestReflectionFuncName(t *testing.T) {
	funOptionType := reflect.TypeOf(gosh.FunOption(func(fo *gosh.FunOptions) {}))

	dep := gosh.Dep("foo")
	depType := reflect.TypeOf(dep)

	v := depType.AssignableTo(funOptionType)
	if !v {
		t.Errorf("v=%v", v)
	}
}

func RetStr(m string) string {
	return "ret" + m
}

func RetStrMap(k, v string) map[string]string {
	return map[string]string{k: v}
}

func TestReflectCallToReturnStr(t *testing.T) {
	f := reflect.ValueOf(RetStr)

	rets := f.Call([]reflect.Value{reflect.ValueOf("foo")})

	assert.Equal(t, rets[0].String(), "retfoo")
}

func TestReflectCallToReturnStrMap(t *testing.T) {
	f := reflect.ValueOf(RetStrMap)

	rets := f.Call([]reflect.Value{reflect.ValueOf("foo"), reflect.ValueOf("bar")})

	assert.Equal(t, rets[0].MapIndex(reflect.ValueOf("foo")).String(), "bar")
}
