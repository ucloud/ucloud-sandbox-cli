package benchmark

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
)

type lifecycleOperation struct {
	runOptions

	exec     execOptions
	skipExec bool
}

func (o *lifecycleOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "lifecycle",
		Short: "Measure sandbox create, first exec and kill latency",
		Args:  cobra.NoArgs,
	}

	o.addFlags(c, "Seconds each sandbox lives for, bounding any sandbox a failed kill leaves behind")
	c.Flags().StringVarP(&o.exec.command, "cmd", "", "echo ok", "Command run in each sandbox during the exec phase")
	c.Flags().StringVarP(&o.exec.user, "user", "u", "", "User to run the exec command as")
	c.Flags().BoolVarP(&o.skipExec, "skip-exec", "", false, "Skip the exec phase, measuring create and kill only")

	return c
}

func (o *lifecycleOperation) validate() error {
	if err := o.runOptions.validate(); err != nil {
		return err
	}
	if !o.skipExec && o.exec.command == "" {
		return fmt.Errorf("--cmd cannot be empty; use --skip-exec to leave out the exec phase")
	}
	return nil
}

func (o *lifecycleOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}
	return o.run(ctx, "lifecycle", o.exec, benchmark.LifecycleOptions{SkipExec: o.skipExec})
}
