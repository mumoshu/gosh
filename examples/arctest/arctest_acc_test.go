package arctest_test

import (
	"bytes"
	"testing"

	"github.com/mumoshu/gosh"
	"github.com/mumoshu/gosh/examples/arctest"
	"github.com/mumoshu/gosh/goshtest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var tt *testing.T
var arctestSh *gosh.Shell

func TestAcc(t *testing.T) {
	arctestSh = arctest.New()

	goshtest.Run(t, arctestSh, func() {
		tt = t
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

		args = append(args, tt)
		args = append(args, config.cmd)
		args = append(args, config.args...)
		args = append(args, "-dry-run", "-test-id=abcdefg")
		args = append(args, gosh.WriteStdout(&stdoutBuf), gosh.WriteStderr(&stderrBuf))

		err = arctestSh.Run(args...)

		stdout = stdoutBuf.String()
		stderr = stderrBuf.String()
	})

	Describe("e2e", func() {
		BeforeEach(func() {
			config.cmd = "e2e"
		})

		Context("default", func() {
			BeforeEach(func() {
				config.args = []interface{}{}
			})

			It("should output \"hello world\"", func() {
				Expect(stdout).To(Equal("hello world\n"))
			})

			It("should write \"hello world (stderr)\" to stderr", func() {
				Expect(stderr).To(Equal("Using workdir at .e2e/workabcdefg\nhello world (stderr)\nDeleting cluster \"workabcdefg\" ...\n"))
			})

			It("should not error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
})
