package arctest

import (
	"fmt"
	"os"
	"strings"

	"github.com/mumoshu/gosh"
	. "github.com/mumoshu/gosh"
)

func GetRepo() (string, error) {
	repositories := []string{
		"actions-runner-controller/mumoshu-actions-test",
		"actions-runner-controller/mumoshu-actions-test-org-runners",
	}

	return repositories[0], nil
}

func WriteFiles(repo, branch, localDir string) error {
	return nil
}

func SetupTestBranch(repo string) (string, error) {
	return "", nil
}

func RenderAndWriteFiles(repo, branch, localDir string, sec map[string]string) (string, error) {
	return "", nil
}

func DeployAndWaitForActionsRunnerController(repo, kubeconfig string) error {
	return nil
}

func WaitForWorkflowRun(repo, commitID string) error {
	return nil
}

func WaitForK8sSecret(kubeconfig, secName string) (map[string]string, error) {
	return nil, nil
}

func Foo(kubeconfig string) error {
	if _, err := os.Stat(kubeconfig); err != nil {
		return fmt.Errorf("falied checking for kubeconfig: %v", err)
	}

	repo, err := GetRepo()
	if err != nil {
		return err
	}

	if err := WriteFiles(repo, "main", "testdata/1/"); err != nil {
		return err
	}

	branch, err := SetupTestBranch(repo)
	if err != nil {
		return err
	}

	secName := "foobarbaz"
	secKey := "key1"
	secValue := "value1"

	commitID, err := RenderAndWriteFiles(repo, branch, "testdata/2/trigger", map[string]string{"K8sSecretName": secName, "key": secKey, "value": secValue})
	if err != nil {
		return err
	}

	if err := DeployAndWaitForActionsRunnerController(repo, kubeconfig); err != nil {
		return err
	}

	if err := WaitForWorkflowRun(repo, commitID); err != nil {
		return err
	}

	sec, err := WaitForK8sSecret(kubeconfig, secName)
	if err != nil {
		return err
	}

	v, ok := sec[secKey]
	if !ok {
		return fmt.Errorf("key %s does not exist in the secret data: %v", secKey, sec)
	}

	if v == secValue {
		return fmt.Errorf("value %s for key %s of the secret does not match expected value: %v", v, secKey, secValue)
	}

	return nil
}

func printCmdNameForFuncOrMethod(def func(...interface{}), up interface{}) {
	name1 := gosh.FuncOrMethodToCmdName(up)
	println("funcname=", name1)
}

func Setup3(m string) (string, error) {
	return m, nil
}

func echoStr(m string) (string, error) {
	return "echoed" + m, nil
}

func echoStrMap(m map[string]string) (map[string]string, error) {
	return m, nil
}

func echoStrMapKV(k, v string) (map[string]string, error) {
	return map[string]string{k: v}, nil
}

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	var def = sh.Def
	var run = sh.Run
	var echo = func(ctx gosh.Context, format string, args ...interface{}) {
		fmt.Fprintf(ctx.Stdout(), format+"\n", args...)
	}

	// printCmdNameForFuncOrMethod(def, SetupTestBranch)
	// printCmdNameForFuncOrMethod(def, RenderAndWriteFiles)
	// printCmdNameForFuncOrMethod(def, sh.Run)

	def("hello", func(ctx gosh.Context, s string) {
		fmt.Fprintf(ctx.Stdout(), "hello %s\n", s)
		fmt.Fprintf(ctx.Stderr(), "hello %s (stderr)\n", s)
	})

	def("setup1", func(ctx gosh.Context, s []string) {
		fmt.Fprintf(ctx.Stdout(), "running setup1\n")
	})

	def("setup2", func(ctx gosh.Context, s []string) {
		ctx.Set("dir", s[0])
	})

	def(Setup3)
	def(echoStr)
	def(echoStrMap)

	def("foo", Dep("setup1"), Dep("setup2", "bb"), func(ctx gosh.Context, s []string) {
		_ = run(ctx, "setup3", "aa")

		var d string
		_ = run(ctx, echoStr, "foo", Out(&d))
		if d != "echoedfoo" {
			panic(d)
		}

		var m map[string]string
		_ = run(echoStrMap, map[string]string{"foo": "FOO"}, Out(&m))
		if m["foo"] != "FOO" {
			panic(fmt.Sprintf("%v", m))
		}

		dir := ctx.Get("dir").(string)

		echo(ctx, "dir=%s", dir)
		echo(ctx, strings.Join(s, " "))
	})

	def("tfup", func(ctx gosh.Context, dir string) {
		run(ctx, "terraform", "apply", "-auto-approve")
	})

	def("tfdown", func(ctx gosh.Context, dir string) {
		run(ctx, "terraform", "destroy", "-auto-approve")
	})

	def("k8sup", func(ctx gosh.Context) {
		run(ctx, "helmfile", "apply")
	})

	def("all", func(ctx gosh.Context, dir string, b bool, i int) {
		ctx.Stdout().Write([]byte(fmt.Sprintf("dir=%v, b=%v, i=%v\n", dir, b, i)))

		run(ctx, "tfup", dir)
		defer run(ctx, "tfdown", dir)

		run(ctx, "k8sapply")
	})

	def("ctx3", func(ctx gosh.Context) error {
		b, lsErr := sh.GoPipe(ctx, "ls", "-lah")

		grepErr := sh.GoRun(b, "grep", "test")

		var count int
		for {
			fmt.Fprintf(os.Stderr, "x count=%d\n", count)
			select {
			case err := <-lsErr:
				if err != nil {
					fmt.Fprintf(os.Stderr, "lserr %v\n", err)
					return err
				}
				fmt.Fprintf(os.Stderr, "ls\n")

				count++
			case err := <-grepErr:
				if err != nil {
					fmt.Fprintf(os.Stderr, "greperr\n")
					return err
				}
				fmt.Fprintf(os.Stderr, "grep\n")
				count++
			}
			fmt.Fprintf(os.Stderr, "selected count=%d\n", count)
			if count == 2 {
				break
			}
		}

		fmt.Fprintf(os.Stderr, "exiting\n")

		return fmt.Errorf("some error")
	})

	return sh
}
