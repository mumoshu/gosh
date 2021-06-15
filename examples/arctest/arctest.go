package arctest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mumoshu/gosh"
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

func New() *gosh.Shell {
	sh := &gosh.Shell{}

	var Echof = func(ctx gosh.Context, format string, args ...interface{}) {
		fmt.Fprintf(ctx.Stdout(), format+"\n", args...)
	}

	type Config struct {
		Region string `flag:"region"`
	}

	sh.Export("terraform", func(ctx gosh.Context, cmd string, args []string) {
		Echof(ctx, "cmd=%s, args=%v", cmd, args)
	})

	sh.Export("terraform-apply", func(ctx gosh.Context, dir string) {
		sh.Run(ctx, "terraform", "apply", "-auto-approve")
	})

	sh.Export("terraform-destroy", func(ctx gosh.Context, dir string) {
		sh.Run(ctx, "terraform", "destroy", "-auto-approve")
	})

	sh.Export("deploy", func(ctx gosh.Context) {
		sh.Run(ctx, "./scripts/deploy.sh")
	})

	sh.Export("test", func(ctx gosh.Context) {

	})

	const ActionsRunnerControllerPath = "~/actions-runner-controller"

	type Opts struct {
		SkipClean bool   `flag:"skip-clean"`
		DryRun    bool   `flag:"dry-run"`
		TestID    string `flag:"test-id"`
	}

	infof := func(ctx gosh.Context, format string, args ...interface{}) {
		fmt.Fprintf(ctx.Stderr(), format+"\n", args...)
	}

	sh.Export("e2e", func(ctx gosh.Context, opts Opts) error {
		if err := os.MkdirAll(".e2e", 0755); err != nil {
			return err
		}

		var workDir string

		if opts.TestID != "" {
			workDir = filepath.Join(".e2e", "work"+opts.TestID)
			if err := os.MkdirAll(workDir, 0755); err != nil {
				return err
			}
		} else {
			var err error

			workDir, err = os.MkdirTemp(".e2e", "work")
			if err != nil {
				return err
			}
		}
		infof(ctx, "Using workdir at %s", workDir)
		defer func() {
			if opts.SkipClean {
				infof(ctx, "Skipped removing %s", workDir)
				return
			}
			if workDir == "" || workDir == "/" || workDir == "." {
				return
			}
			os.RemoveAll(workDir)
		}()

		name := filepath.Base(workDir)

		kubeconfigPath := filepath.Join(workDir, "kubeconfig")
		kubeconfigEnv := gosh.Env(fmt.Sprintf("%s=%s", "KUBECONFIG", kubeconfigPath))

		if !opts.DryRun {
			if err := sh.Run(ctx, kubeconfigEnv, "kind", "create", "cluster", "--name", name); err != nil {
				return err
			}
		}
		defer func() {
			if opts.SkipClean {
				infof(ctx, "Skipped `kind delete cluster --name %s`", name)
				return
			}
			sh.Run(ctx, "kind", "delete", "cluster", "--name", name)
		}()

		if !opts.DryRun {
			if err := sh.Run(ctx, kubeconfigEnv, "kubectl", "get", "node"); err != nil {
				return err
			}
		}

		ctx.Stdout().Write([]byte("hello " + "world" + "\n"))
		ctx.Stderr().Write([]byte("hello " + "world" + " (stderr)\n"))

		sh.Run("terraform-apply", "foo")
		defer sh.Run("terraform-destroy", "foo")

		sh.Run("deploy")

		sh.Run("test")

		return nil
	})

	return sh
}
