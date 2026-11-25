package benchmark

import (
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
)

// createOperation times sandbox creation alone. The sandboxes are not killed:
// each lives until its timeout reclaims it, so the number running grows with
// every iteration.
type createOperation struct {
	runOptions
}

func (o *createOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Measure sandbox create latency, leaving the sandboxes to time out",
		Args:  cobra.NoArgs,
	}

	o.addFlags(c, "Seconds each sandbox lives for before it is reclaimed; sandboxes are not killed")

	return c
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}
	return o.run(ctx, "create", execOptions{}, benchmark.LifecycleOptions{SkipExec: true, SkipKill: true})
}
