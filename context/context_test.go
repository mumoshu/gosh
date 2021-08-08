package context_test

import (
	"github.com/mumoshu/gosh/context"

	gocontext "context"
	"testing"
)

func TestGoContextInterop(t *testing.T) {
	f := func(_ context.Context) {

	}

	f(gocontext.TODO())
	f(context.WithError(gocontext.TODO(), nil))

	g := func(_ gocontext.Context) {

	}

	g(context.TODO())
	f(context.WithError(context.TODO(), nil))
}
