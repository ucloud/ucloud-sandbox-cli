package fs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/files"
)

// Command returns the fs command group, a subcommand of sandbox.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "fs",
		Short: "Manage a sandbox's filesystem",
	}
	c.AddCommand(cmd.Build(&catOperation{}))
	c.AddCommand(cmd.Build(&cpOperation{}))
	c.AddCommand(cmd.Build(&lsOperation{}))
	c.AddCommand(cmd.Build(&mkdirOperation{}))
	c.AddCommand(cmd.Build(&mvOperation{}))
	c.AddCommand(cmd.Build(&rmOperation{}))
	return c
}

// session is what every fs command shares: the user the operation runs as, and
// the two steps that turn a sandbox ID into a filesystem to work on.
type session struct {
	user string
}

// AddFlags registers the flag picking the user the operation runs as. envd
// needs one to resolve a relative path and to decide what the operation is
// allowed to touch, so every fs command takes it.
func (s *session) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.user, "user", "u", "", "User to run the operation as")
}

// open connects to the sandbox and returns its filesystem.
func (s *session) open(ctx cmd.OperationContext, sandboxID string) (*files.Filesystem, error) {
	// Connecting resumes a paused sandbox, so its files can be reached either
	// way.
	sbx, err := ctx.Client.Sandboxes().Connect(ctx, sandboxID, api.ConnectSandbox{})
	if err != nil {
		return nil, err
	}

	envd, err := ctx.Client.Sandboxes().Envd(sbx, s.user)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sandbox envd: %w", err)
	}

	return envd.Files(), nil
}
