package snapshot

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type createOperation struct {
	req api.SandboxSnapshotRequest
}

func (o *createOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "create <sandbox-id>",
		Aliases: []string{"cr"},
		Short:   "Snapshot a sandbox",
		Args:    cobra.ExactArgs(1),
	}

	flags.NullableStringVarP(c.Flags(), &o.req.Name, "name", "n",
		"Name for the snapshot; reusing one adds a build to that snapshot instead of making a new one")

	return c
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	snapshot, err := ctx.Client.Sandboxes().CreateSnapshot(ctx, ctx.Args[0], o.req)
	if err != nil {
		return err
	}

	// The ID is what `sandbox create` takes as its template, which is how a
	// snapshot is booted again.
	fmt.Printf("Snapshot created: %s\n", snapshot.SnapshotID)

	return nil
}
