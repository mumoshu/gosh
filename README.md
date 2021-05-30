# gosh

`gosh` is a framework for operating and extending Linux shells with Go.

`gosh` makes it extremely easy to gradually rewrite your complex shell script into a more maintainable equivalent.

You would usually use it to:

- Build your project, as an alternative to `make`
- Write Bash functions in Go
- Build your own shell with custom functions written in Go
- Incrementally refactor your Bash scripts with Go
- Test your Bash scripts, as a dependency injection system and a test framework/runner

Features:

- [Interactive Shell with Hot Reloading](#interactive-shell-with-hot-reloading)
- [Use as a Build Tool](#use-as-a-build-tool)
- [Diagnostic Logging](#diagnostic-logging)
- [`go test` Integration](#go-test-integration)
- [Ginkgo Integration](#ginkgo-integration)

`gosh` primarily targets `bash` but it can be easily enhanced to support other shells.
Any contributions to add more shell supports are always welcomed.

## Usage

Get started by writing a Go application provides a shell that has a built-in `hello` function:

```
$ mkdir ./myapp; cat <<EOS > ./myapp/main.go
package main

import (
	"os"

	"github.com/mumoshu/gosh"
)

func main() {
	sh := &gosh.Shell{}

	sh.Def("hello", func(ctx gosh.Context, target string) {
		ctx.Stdout().Write([]byte("hello " + target + "\n"))
	})

	sh.MustExec(os.Args)
}
EOS
```

Go-running it takes you into a shell session that has the builtin function:

```
$ go run ./myapp

bash$ hello world
hello world
```

This shell session rebuilds your command automatically so that you can test the modified version of `hello` function without restarting the session.

Once you're confident with what you built, you'd want to distribute it.

Just use a standard `go build` command to create a single executable that provides a `bash` enviroment that provides custom functions:

```
$ go build -o myapp ./myapp
```

You can directly invoke the custom function by providing the command name and arguments like:

```
$ myapp hello world
```

```
hello world!
```

As it's a bash in the end, you can run a shell script, with access to the `hello` function written in Go:

```
$ myapp <<EOS
for ((i=0; i<3; i++); do
  hello world
done
EOS
```

```
hello world!
hello world!
hello world!
```

The custom functions and the shell environemnt can be used to not only accelerating your application, but also to mock your commands for testing. The possibilities are endless. If you're curious, read on.

# Features

## Custom Functions written in Go

You can `go run` your app to call a single custom function, or a shell script that invokes the custom function.

Our [`getting-started` example](./run/examples/getting-started) defines a single custom function `hello` that takes only one argument to specify the part of the hello-world message printed by it.

So, `hello world` should print `hello world`.

To invoke it directly, just provide the command and the args via the command-line.

With `go run`, you should write:

```
$ go run ./run/examples/getting-started hello world
hello world
```

It can be executed the same way against a prebuilt binary:

```
$ go build -o getting-started ./run/examples/getting-started
$ ./getting-started hello world
hello world
```

It works exactly like a standard shell equipped with custom functions.

As similar as you can provide shell scripts to `bash` via the standard input, you can do the same on your command:

```
$ go run ./run/examples/getting-started <<EOS
for ((i=0; i<3; i++)); do
hello world
done
EOS
```

```
hello world
hello world
hello world
```

```
$ cat <<EOS > test.gosh
for ((i=0; i<3; i++)); do
hello world
done
EOS
```

You can use just point the script file or use redirection to source the script to run:

```
$ go run ./run/examples/getting-started test.gosh
$ go run ./run/examples/getting-started <test.gosh
```

## Interactive Shell with Hot Reloading

You can `go run` your app without any arguments to start an interactive shell that hot reloads the custom functions automatically:

```
$ go run ./run/examples/getting-started
```

Once the new interactive shell session gets started, you use it like a regular shell.

The `getting-started` example contains a custom function written in Go, named `hello`, that prints `hello <FIRST ARG>` to the standard output.

```golang
sh.Def("hello", func(ctx gosh.Context, target string) {
    ctx.Stdout().Write([]byte("hello " + target + "\n"))
})
```

To invoke `hello`, refer to it like a standard shell function:

```
gosh$ hello world
hello world
```

The interactive shell hot-reloads your Go code.
That is, you can modify the custom function code and just reinvoke `hello`, without restarting the shell.

For example, we modify the code so that the custom function prints it without another prefix `konnichiwa`:

```
$ code ./run/examples/getting-started/main.go
```

```golang
sh.Def("hello", func(ctx gosh.Context, target string) {
    ctx.Stdout().Write([]byte("konnichiwa " + target + "\n"))
})
```

Go back to your termiinal and rerun `hello world` to see the code hot-reloaded:

```
gosh$ hello world
konnichiwa world
```

## Use as a Build Tool

As you can seen in our [`project` example](project/build.go), `gosh` has a few utilities to help
using your `gosh` application as a build tool like `make`.

Let's say you previously had a `Makefile` that looked like this:

```
.PHONY: all build test

all: build test

build:
    go build -o getting-started ./run/examples/getting-started

test:
    go test ./...
```

You can rewrite it by using some Go code powered by `gosh` that looks like the below:

```
Def("all", Dep("build"), Dep("test"), func() {

})

Def("build", func() {
    Run("go", "build", "-o", "getting-started", "./run/examples/getting-started")
})

Def("test", func() {
    Run("go", "test", "./...")
})
```

Instead of `make all`, `make build`, and `make test` you used to run, you can now run respective `go run` commands:

```
# Runs a go build
$ go run -tags=project ./project build

# Runs a go test
$ go run -tags=project ./project test

# runs test and build
$ go run -tags=project ./project <<EOS
test
build
EOS

# Runs all
# go run -tags=project ./project all
```

I'd personally recommend you to have a short alias, so that rerunning it becomes very easy:

```
alias project='go run -tags=project ./project'

# Runs go build
project build

# Runs go test
project test

@ Runs test and build
project <<EOS
test
build
EOS

# Runs all
project all
```

An extra care needs to be taken if you want to run it interactively while using a Go build tag.

As [Go has no way or plan to expose the build tag at runtime](https://github.com/golang/go/issues/7007#issuecomment-66089610), you need to use another way,
a dedicated environment variable implemented by `gosh`, to tell it to use the build tag for hot-reloading.

Otherwise, you get an go-build error like `package github.com/mumoshu/gosh/project: build constraints exclude all Go files in /home/mumoshu/p/gosh/project`.

The environment variable is `GOSH_BUILD_TAG`, and you should set it like:

```
GOSH_BUILD_TAG=project project

# Or include it in the alias...

alias project='GOSH_BUILD_TAG=project go run -tags=project ./project'

project
```

## Diagnostic Logging

In case you aren't sure why your custom shell functions and the whole application doesn't work,
try reading diagnostic logs that contains various debugging information from `gosh`.

To access the diagnostic logs, use the file descriptor `3` to redirect it to an arbitrary destination:

```
$ go run ./run/examples/getting-started hello world 3>diags.out

$ cat diags.out
2021-05-29T05:59:38Z    app.go:466      registering func hello
```

You can also emit your own diagnostic logs from your custom functions, standard shell functions, shell snippets, and even another application written in a totally different progmming language.

If you want to write a diagnostic log message from a custom function written in Go, use the `gosh.Shell.Diagf` function:

```
$ code ./run/examples/getting-started/main.go
```

```
sh.Def("hello", func(ctx gosh.Context, target string) {
    // Add the below Diagf call
    sh.Diagf("My own debug message someData=%s someNumber=%d", "foobar", 123)

    ctx.Stdout().Write([]byte("hello " + target + "\n"))
})
```

```
$ go run ./run/examples/getting-started hello world 3>diags.out

$ cat diags.out 
2021-05-29T06:03:58Z    app.go:466      registering func hello
2021-05-29T06:03:58Z    main.go:13      My own debug message someData=foobar someNumber=123
```

It also works with a script, without any surprise:

```
$ go run ./run/examples/getting-started <<EOS 3>diags.out
hello world
hello world
EOS

$ cat diags.out 
2021/05/29 06:16:57 <nil>
2021-05-29T06:16:54Z    app.go:466      registering func hello
2021-05-29T06:16:54Z    app.go:466      registering func hello
2021-05-29T06:16:54Z    app.go:466      registering func hello
2021-05-29T06:16:54Z    main.go:13      My own debug message someData=foobar someNumber=123
2021-05-29T06:16:55Z    app.go:466      registering func hello
2021-05-29T06:16:55Z    app.go:466      registering func hello
2021-05-29T06:16:55Z    main.go:13      My own debug message someData=foobar someNumber=123
```

To write a log message from a shell script, just write to fd 3 using a standard shell syntax.

In Bash, you use `>&3` like:

```bash
echo my own debug message >&3
```

To test, you can e.g. Bash [`Here Strings`](https://www.gnu.org/software/bash/manual/html_node/Redirections.html#Here-Strings):

```
$ go run ./run/examples/getting-started <<EOS 3>diags.out
echo my own debug message >&3
EOS

$ cat diags.out 
2021-05-29T06:08:19Z    app.go:466      registering func hello
2021-05-29T06:08:19Z    app.go:466      registering func hello
my own debug message
```

For a bash function, it is as easy as...:

```
$ go run ./run/examples/getting-started <<EOS 3>diags.out
myfunc() {
  echo my own debug message from myfunc >&3
}

myfunc
EOS

$ cat diags.out 
2021-05-29T06:14:09Z    app.go:466      registering func hello
2021-05-29T06:14:09Z    app.go:466      registering func hello
my own debug message from myfunc
```

## `go test` Integration

See the [gotest example](run/examples/gotest/gotest_test.go) for how to write unit and integration tests against your `gosh` applications, shell scripts, or even a regular command that isn't implemented using `gosh`.

The below is the most standard structure of an unit test for your `gosh` application:

```
func TestUnit(t *testing.T) {
	gotest := gotest.New()

	var stdout bytes.Buffer

	if err := gotest.Run("hello", "world", gosh.WriteStdout(&stdout)); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "hello world\n", stdout.String())
}
```

In the above example, `gotest.New` is implemented by you to provide an instance of `*gosh.Shell`. You write tests against it by calling the `Run` function, using some helper like `gosh.WriteStdout` to capture what's written to the standard output by your application.

If you are curious how you would implement `gotest.New`, read on.

### Structuring your gosh application for ease of testing

A recommended approach to structure your `gosh` application is to put everything except the entrypoint to your application to a dedicated package.

In the `gotest example`, we have two packages:

- `gotest/cmd` that contains the `main` package
- `gotest` for everything else

The only source file that exists in the first package is `gotest/cmd/main.go`.

Basically, It contains only a call to the second package:

```
package main

func main() {
	gotest.MustExec(os.Args)
}
```

The `gotest.MustExec` call refers to the `MustExec` function defined in the second package.

It looks like:

```
package gotest

func MustExec(osArgs []string) {
	New().MustExec(osArgs)
}
```

`New` is the function that initializes the instance of your `gosh` application:

```
package gotest

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	sh.Def("hello", func(ctx gosh.Context, target string) {
		ctx.Stdout().Write([]byte("hello " + target + "\n"))
	})

	return sh
}
```

This way, you can just call `New` to create an instance of your gosh application for testing, and then call `Run` on the application in the test, as you would do while writing the application itself.

## Ginkgo Integration

If you find yourself repeating a lot of "test setup" code in your tests or have hard time structuring your tests against a lot of cases, you might find [Ginkgo](https://onsi.github.io/ginkgo/) helpful.

> Ginkgo is a Go testing framework built to help you efficiently write expressive and comprehensive tests using Behavior-Driven Development (“BDD”) style
> https://onsi.github.io/ginkgo/

See the [ginkgotest example](run/examples/ginkgotest/ginkgotest_test.go) for how to write integration and End-to-End tests against your `gosh` applications, shell scripts, or even a regular command that isn't implemented using `gosh`.

The below is the most standard structure of an Ginkgo test for your `gosh` application:

```
var app *gosh.Shell

func TestAcc(t *testing.T) {
	app = ginkgotest.New()

	app.In(t, func() {
		RegisterFailHandler(Fail)
		RunSpecs(t, "Your App's Suite")
	})
}

var _ = Describe("Your App", func() {
	var (
		config struct {
			cmd  string
			args []interface{}
		}

		err    error
		stdout string
	)

	JustBeforeEach(func() {
		var stdoutBuf bytes.Buffer

		var args []interface{}

		args = append(args, config.cmd)
		args = append(args, config.args...)
		args = append(args, gosh.WriteStdout(&stdoutBuf))

		err = app.Run(args...)

		stdout = stdoutBuf.String()
	})

	Describe("hello", func() {
		BeforeEach(func() {
			config.cmd = "hello"
		})

		Context("world", func() {
			BeforeEach(func() {
				config.args = []interface{}{"world"}
			})

			It("should output \"hello world\"", func() {
				Expect(stdout).To(Equal("hello world\n"))
			})
		})
    })
})
```

In the above example, `gotest.New` is implemented by you to provide an instance of `*gosh.Shell`. You write tests against it by calling the `Run` function, using some helper like `gosh.WriteStdout` to capture what's written to the standard output by your application.

If you are curious how you would implement `gotest.New`, read [Structuring your gosh application for ease of testing](#structuring-your-gosh-application-for-ease-of-testing).

The followings are standard functions provided by Ginkgo:

- RegisterFailHandler
- RunSpecs
- Describe
- JustBeforeEach / BeforeEach
- It

The followings are standard functions provided by Gomega, which is Ginkgo's preferred test helpers and matchers library.

- Expect
- Equal

Please refer to [Ginkgo's official documentation](https://onsi.github.io/ginkgo/) for knowing what each Ginkgo and Gomega functions mean and how to write Ginkgo test scenarios.

# Acknowledgements

`gosh` has been inspired by numerous open-source projects listed below.

Much appreciation to the authors and the open-source community!

Task runners:

- https://github.com/magefile

*unix pipeline-like things:

- https://github.com/b4b4r07/go-pipe
- https://github.com/go-pipe/pipe
- https://github.com/urjitbhatia/gopipe
- https://github.com/mattn/go-pipeline

Misc:

- https://github.com/taylorflatt/remote-shell
- https://github.com/hashicorp/go-plugin
- https://github.com/a8m/reflect-examples
- https://medium.com/swlh/effective-ginkgo-gomega-b6c28d476a09
- https://medium.com/@william.la.martin/ginkgotchas-yeh-also-gomega-13e39185ec96
- https://github.com/fsnotify/fsnotify

FD handling:

- https://gist.github.com/miguelmota/4ac6c2c127b6853593808d9d3bba067f
- https://stackoverflow.com/questions/7082001/how-do-file-descriptors-work
