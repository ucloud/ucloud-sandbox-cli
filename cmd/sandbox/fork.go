package sandbox

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type forkOperation struct {
	req api.SandboxForkRequest

	dryRun bool
}

func (o *forkOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "fork <sandbox-id>",
		Aliases: []string{"fk"},
		Short:   "Fork a sandbox",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableInt32VarP(c.Flags(), &o.req.Count, "count", "", "Number of forks to create")
	flags.NullableInt32VarP(c.Flags(), &o.req.Timeout, "timeout", "", "Seconds after which the forks expire")

	c.Flags().BoolVarP(&o.dryRun, "dry-run", "", false, "Show fork request")

	return c
}

func (o *forkOperation) Run(ctx cmd.OperationContext) error {
	if o.dryRun {
		return cmd.ShowJSON(o.req)
	}

	results, err := ctx.Client.Sandboxes().Fork(ctx, ctx.Args[0], o.req)
	if err != nil {
		return err
	}

	// Every requested fork succeeds or fails on its own, so all of them are
	// reported: one line per fork, so no created sandbox goes unnamed.
	failed := 0
	for _, result := range results {
		if result.Sandbox != nil {
			fmt.Printf("Sandbox forked: %s\n", result.Sandbox.SandboxID)
			continue
		}

		failed++

		message := "fork failed"
		if result.Error != nil {
			message = result.Error.Message
		}
		fmt.Fprintf(os.Stderr, "Fork failed: %s\n", message)
	}

	if failed > 0 {
		return fmt.Errorf("%d of %d forks failed", failed, len(results))
	}

	return nil
}
