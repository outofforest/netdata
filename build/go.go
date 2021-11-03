package build

import (
	"context"
	"os"
	"os/exec"

	"github.com/wojciech-malota-wojcik/build"
	"github.com/wojciech-malota-wojcik/netdata/lib/run"
)

func goBuildPkg(ctx context.Context, pkg, out string, cgo bool) error {
	cmd := exec.Command("go", "build", "-o", out, "./"+pkg)
	if !cgo {
		cmd.Env = append([]string{"CGO_ENABLED=0"}, os.Environ()...)
	}
	return run.Exec(ctx, cmd)
}

func goModTidy(ctx context.Context) error {
	return run.Exec(ctx, exec.Command("go", "mod", "tidy"))
}

func goLint(ctx context.Context, deps build.DepsFunc) error {
	if err := run.Exec(ctx, exec.Command("golangci-lint", "run", "--config", "build/.golangci.yaml")); err != nil {
		return err
	}
	deps(goModTidy, gitStatusClean)
	return nil
}

func goImports(ctx context.Context) error {
	return run.Exec(ctx, exec.Command("goimports", "-w", "."))
}

func goTest(ctx context.Context) error {
	return run.Exec(ctx, exec.Command("go", "test", "-count=1", "-race", "./..."))
}
