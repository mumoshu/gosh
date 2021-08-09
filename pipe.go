package gosh

import (
	"io"
	"log"

	"github.com/mumoshu/gosh/context"
)

func (sh *Shell) GoPipe(ctx context.Context, vars ...interface{}) (context.Context, <-chan error) {
	a, b, close := sh.PipeFromContext(ctx)

	err := make(chan error)

	go func() {
		vars = append([]interface{}{a}, vars...)
		e := sh.Run(vars...)
		close()
		err <- e
	}()

	return b, err
}

func (sh *Shell) PipeFromContext(ctx context.Context) (context.Context, context.Context, func()) {
	a, b := context.Background(), context.Background()

	r, w := io.Pipe()

	a = context.WithStdin(a, context.Stdin(ctx))
	a = context.WithStdout(a, w)
	a = context.WithStderr(a, context.Stderr(ctx))

	b = context.WithStdin(b, r)
	b = context.WithStdout(b, context.Stdout(ctx))
	b = context.WithStderr(b, context.Stderr(ctx))

	return a, b, func() {
		if err := w.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
