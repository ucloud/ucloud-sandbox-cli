package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
)

type Operation interface {
	Command() *cobra.Command
	Run(ctx OperationContext) error
}

type OperationContext struct {
	context.Context

	Cmd    *cobra.Command
	Args   []string
	Client *client.Client
	Config *config.Config
}

func Build(op Operation) *cobra.Command {
	cmd := op.Command()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client, err := config.NewClient(cfg)
		if err != nil {
			return err
		}
		ctx := OperationContext{
			Context: cmd.Context(),

			Cmd:    cmd,
			Args:   args,
			Client: client,
			Config: cfg,
		}
		return op.Run(ctx)
	}
	return cmd
}

// BuildLocal wraps an operation that only touches the local configuration:
// these commands run before there are credentials to build a client from, or
// exist to repair the ones that are there.
//
// The context carries neither a client nor a config; an operation that needs
// the config reads it itself, so a file that fails to parse cannot stop the
// command that opens it for editing.
func BuildLocal(op Operation) *cobra.Command {
	cmd := op.Command()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return op.Run(OperationContext{
			Context: cmd.Context(),

			Cmd:  cmd,
			Args: args,
		})
	}
	return cmd
}

func ShowJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
