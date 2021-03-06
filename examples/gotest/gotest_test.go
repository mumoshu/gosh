package gotest_test

import (
	"bytes"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/examples/gotest"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestUnitSmoke(t *testing.T) {
	gotest := gotest.New()

	goshtest.Run(t, gotest, func() {
		if err := gotest.Run(t, "hello", "world"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestUnit(t *testing.T) {
	gotest := gotest.New()

	goshtest.Run(t, gotest, func() {
		var stdout bytes.Buffer

		if err := gotest.Run(t, "hello", "world", gosh.WriteStdout(&stdout)); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "hello world\n", stdout.String())
	})
}

func TestIntegration(t *testing.T) {
	sh := gotest.New()

	goshtest.Run(t, sh, func() {
		var stdout bytes.Buffer

		err := sh.Run(t, "bash", "-c", "for ((i=0;i<3;i++)); do hello world; done", gosh.WriteStdout(&stdout))

		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "hello world\nhello world\nhello world\n", stdout.String())
	})
}
