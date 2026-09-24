package sandbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

type killOperation struct {
	// params narrows which sandboxes --all picks up.
	params api.SandboxListParamsV2

	all   bool
	yes   bool
	limit int
}

func (o *killOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "kill [sandbox-id...]",
		Aliases: []string{"kl"},
		Short:   "Kill one or more sandboxes",
		Args:    cobra.ArbitraryArgs,
	}

	sandboxListParamsVarP(c.Flags(), &o.params)

	c.Flags().BoolVarP(&o.all, "all", "a", false, "Kill every sandbox matching the filters")
	c.Flags().BoolVarP(&o.yes, "yes", "y", false, "Don't ask for confirmation")
	c.Flags().IntVarP(&o.limit, "limit", "l", 0, "Maximum number of sandboxes --all kills (0 kills all of them)")

	return c
}

// validate checks the flags against the sandbox IDs given on the command line.
func (o *killOperation) validate(args []string) error {
	if len(args) == 0 && !o.all {
		return fmt.Errorf("specify sandbox IDs or use --all")
	}

	if len(args) > 0 && o.all {
		return fmt.Errorf("cannot use --all together with sandbox IDs")
	}

	if o.limit < 0 {
		return fmt.Errorf("invalid --limit %d: must be 0 or greater", o.limit)
	}

	return nil
}

func (o *killOperation) Run(ctx cmd.OperationContext) error {
	if err := o.validate(ctx.Args); err != nil {
		return err
	}

	ids := ctx.Args

	if o.all {
		matched, err := o.matchingIDs(ctx, ctx.Client.Sandboxes().ListV2(ctx, &o.params))
		if err != nil {
			return err
		}

		if len(matched) == 0 {
			fmt.Println("No sandboxes found.")
			return nil
		}

		// An explicit list of IDs is intent enough; a filter that turns out to
		// match more than expected is not.
		if !o.yes {
			confirmed, err := prompt.Confirm(fmt.Sprintf("Kill %d sandbox(es)", len(matched)))
			if err != nil {
				return err
			}
			if !confirmed {
				return nil
			}
		}

		ids = matched
	}

	// Killing stops at the first failure, so a broken run does not keep tearing
	// sandboxes down. That is also why this is sequential: a request already in
	// flight could not be skipped.
	total := 0
	for _, id := range ids {
		killed, err := ctx.Client.Sandboxes().Kill(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to kill sandbox %s: %w", id, err)
		}

		// A sandbox that is already gone satisfies the intent, so it is
		// reported rather than treated as a failure.
		if !killed {
			fmt.Printf("Sandbox %s not found.\n", id)
			continue
		}

		total++
		fmt.Printf("Sandbox %s killed.\n", id)
	}

	fmt.Printf("Killed %d sandbox(es).\n", total)

	return nil
}

// matchingIDs collects the IDs of the sandboxes the listing yields, stopping
// once limit of them have been found. A limit of zero collects all of them.
func (o *killOperation) matchingIDs(
	ctx context.Context,
	paginator *transport.Paginator[api.ListedSandbox],
) ([]string, error) {
	var ids []string
	for paginator.HasNext() {
		if o.limit > 0 && len(ids) >= o.limit {
			break
		}

		page, err := paginator.NextItems(ctx)
		if err != nil {
			return nil, err
		}

		for _, sbx := range page {
			ids = append(ids, sbx.SandboxID)
		}
	}

	if o.limit > 0 && len(ids) > o.limit {
		ids = ids[:o.limit]
	}

	return ids, nil
}
