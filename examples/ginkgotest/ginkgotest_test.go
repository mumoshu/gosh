package ginkgotest_test

import (
	"bytes"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/examples/ginkgotest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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
		// This doesn't work as then we have no way to "hook" into the test framework
		// for handling indirectly run commands.
		//
		// sh := ginkgotest.New()

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

			It("should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("sekai", func() {
			BeforeEach(func() {
				config.args = []interface{}{"sekai"}
			})

			It("should output \"hello sekai\"", func() {
				Expect(stdout).To(Equal("hello sekai\n"))
			})

			It("should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
