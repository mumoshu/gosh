package pipeline_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/examples/pipeline"
	"github.com/mumoshu/gosh/goshtest"
	"github.com/stretchr/testify/assert"
)

func TestPipeline(t *testing.T) {
	gotest := pipeline.New()

	goshtest.Run(t, gotest, func() {
		var stdout bytes.Buffer

		if err := gotest.Run(t, "ctx3", gosh.WriteStdout(&stdout)); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "footest\n", stdout.String())
	})
}

func TestCancel(t *testing.T) {
	gotest := pipeline.New()

	goshtest.Run(t, gotest, func() {
		var stdout bytes.Buffer

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := gotest.Run(t, ctx, "ctx3", gosh.WriteStdout(&stdout))
		if err == nil {
			t.Fatal("Missing error")
		}

		assert.Equal(t, "context canceled", err.Error())
	})
}
