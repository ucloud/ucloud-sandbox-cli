package sandbox

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type pauseOperation struct {
	req api.SandboxPauseRequest
}

func (o *pauseOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "pause <sandbox-id>",
		Short: "Pause a sandbox, keeping its state for later",
		Args:  cobra.ExactArgs(1),
	}

	flags.NullableBoolVarP(c.Flags(), &o.req.Memory, "memory", "",
		"Capture memory as well as the filesystem; false cold-boots on resume and rules out auto-resume (default true)")

	return c
}

func (o *pauseOperation) Run(ctx cmd.OperationContext) error {
	sandboxID := ctx.Args[0]

	// A sandbox that is already paused is reported as success: the SDK folds
	// the platform's 409 into one, because the intent is satisfied either way.
	if err := ctx.Client.Sandboxes().Pause(ctx, sandboxID, o.req); err != nil {
		return err
	}

	fmt.Printf("Sandbox %s paused.\n", sandboxID)

	return nil
}
