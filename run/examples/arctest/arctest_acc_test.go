package arctest_test

import (
	"bytes"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/run/examples/arctest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var arctestSh *gosh.Shell

func TestAcc(t *testing.T) {
	arctestSh = arctest.New()

	arctestSh.In(t, func() {
		RegisterFailHandler(Fail)
		RunSpecs(t, "Books Suite")
	})
}

var _ = Describe("arctest", func() {
	var (
		config struct {
			cmd  string
			args []interface{}
		}

		err    error
		stdout string
		stderr string
	)

	JustBeforeEach(func() {
		// This doesn't work as then we have no way to "hook" into the test framework
		// for handling indirectly run commands.
		//
		// sh := arctest.New()

		var stdoutBuf, stderrBuf bytes.Buffer

		var args []interface{}

		args = append(args, config.cmd)
		args = append(args, config.args...)
		args = append(args, gosh.WriteStdout(&stdoutBuf), gosh.WriteStderr(&stderrBuf))

		err = arctestSh.Run(args...)

		stdout = stdoutBuf.String()
		stderr = stderrBuf.String()
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

			It("should write \"hello world (stderr)\" to stderr", func() {
				Expect(stderr).To(Equal("hello world (stderr)\n"))
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

			It("should write \"hello sekai (stderr)\" to stderr", func() {
				Expect(stderr).To(Equal("hello sekai (stderr)\n"))
			})

			It("should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
