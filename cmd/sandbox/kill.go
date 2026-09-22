package sandbox

import (
	"context"
	"fmt"
	"sync"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func newKillCmd() *cobra.Command {
	var all bool
	var state string

	cmd := &cobra.Command{
		Use:     "kill [sandbox-id...]",
		Aliases: []string{"kl"},
		Short:   "Kill one or more sandboxes",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && !all {
				return fmt.Errorf("specify sandbox IDs or use --all")
			}
			if len(args) > 0 && all {
				return fmt.Errorf("cannot use --all together with sandbox IDs")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()

			if all {
				return killAll(ctx, client, state)
			}

			var wg sync.WaitGroup
			for _, id := range args {
				wg.Add(1)
				go func(id string) {
					defer wg.Done()
					killOne(ctx, client, id)
				}(id)
			}
			wg.Wait()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Kill all sandboxes")
	cmd.Flags().StringVarP(&state, "state", "s", "running", "Filter by state when used with --all")
	return cmd
}

func killAll(ctx context.Context, client *client.Client, state string) error {
	opts := sandbox.ListV2Options{
		State: []sandbox.State{
			sandbox.State(state),
		},
	}

	paginator := client.Sandboxes().ListV2(ctx, opts)
	total := 0
	for paginator.HasNext() {
		items, err := paginator.NextItems(ctx)
		if err != nil {
			return err
		}
		var wg sync.WaitGroup
		for _, s := range items {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				killOne(ctx, client, id)
			}(s.SandboxID)
		}
		wg.Wait()
		total += len(items)
	}

	if total == 0 {
		fmt.Println("No sandboxes found.")
	} else {
		fmt.Printf("Killed %d sandbox(es).\n", total)
	}
	return nil
}

func killOne(ctx context.Context, client *client.Client, id string) {
	ok, err := client.Sandboxes().Kill(ctx, id)
	if err != nil {
		fmt.Printf("Error killing sandbox %s: %v\n", id, err)
		return
	}
	if !ok {
		fmt.Printf("Sandbox %s not found.\n", id)
		return
	}
	fmt.Printf("Sandbox %s killed.\n", id)
}
