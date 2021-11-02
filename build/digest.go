package build

import (
	"context"
	"os/exec"

	"github.com/wojciech-malota-wojcik/build"
	"github.com/wojciech-malota-wojcik/netdata-digest/lib/run"
)

func buildApp(ctx context.Context) error {
	return goBuildPkg(ctx, "cmd", "bin/client", false)
}

func runApp(ctx context.Context, deps build.DepsFunc) error {
	deps(buildApp)
	return run.Exec(ctx, exec.Command("./bin/client"))
}
