package benchmark

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type lifecycleOperation struct {
	runOptions

	command  string
	user     string
	skipExec bool
}

func (o *lifecycleOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "lifecycle",
		Short: "Measure sandbox create, first exec and kill latency",
		Args:  cobra.NoArgs,
	}

	o.addFlags(c, 10, "Seconds each sandbox lives for, bounding any sandbox a failed kill leaves behind")
	c.Flags().StringVarP(&o.command, "cmd", "", "echo ok", "Command run in each sandbox during the exec phase")
	c.Flags().StringVarP(&o.user, "user", "u", "", "User to run the exec command as")
	c.Flags().BoolVarP(&o.skipExec, "skip-exec", "", false, "Skip the exec phase, measuring create and kill only")

	return c
}

func (o *lifecycleOperation) validate() error {
	if err := o.runOptions.validate(); err != nil {
		return err
	}
	if !o.skipExec && o.command == "" {
		return fmt.Errorf("--cmd cannot be empty; use --skip-exec to leave out the exec phase")
	}
	return nil
}

func (o *lifecycleOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(); err != nil {
		return err
	}
	sdk, err := newClient(ctx.Config)
	if err != nil {
		return err
	}

	spec := runSpec{name: "lifecycle"}
	if !o.skipExec {
		spec.steps = append(spec.steps, execStep(o.command, o.user))
	}
	return o.run(ctx, sdk, spec)
}
