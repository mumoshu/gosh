package gotest_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/run/examples/gotest"
	"github.com/stretchr/testify/assert"
)

func TestUnitSmoke(t *testing.T) {
	gotest := gotest.New()

	if err := gotest.Run("hello", "world"); err != nil {
		t.Fatal(err)
	}
}

func TestUnit(t *testing.T) {
	gotest := gotest.New()

	var stdout bytes.Buffer

	if err := gotest.Run("hello", "world", gosh.WriteStdout(&stdout)); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello world\n", stdout.String())
}

func TestIntegration(t *testing.T) {
	sh := gotest.New()

	sh.In(t, func() {
		fmt.Fprintf(os.Stderr, "%v\n", os.Args)
		var stdout bytes.Buffer

		err := sh.Run(t, "bash", "-c", "for ((i=0;i<3;i++)); do hello world; done", gosh.WriteStdout(&stdout))

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "hello world\nhello world\nhello world\n", stdout.String())
	})
}
