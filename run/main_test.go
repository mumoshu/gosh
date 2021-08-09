package main

import (
	"bytes"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	sh := New()

	goshtest.Run(t, sh, func() {
		t.Run("foo", func(t *testing.T) {
			var stdout bytes.Buffer

			err := sh.Run(t, "foo", "a", "b", gosh.WriteStdout(&stdout))

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "running setup1\ndir=aa\na b\na b\n", stdout.String())
		})
	})

}
