package content

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	volumecontent "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/content"
)

// Command returns the command group for direct volume content access.
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "content",
		Aliases: []string{"ct"},
		Short:   "Manage volume files directly without a sandbox (paths are relative; . is the root)",
	}
	c.AddCommand(cmd.Build(&uploadOperation{}))
	c.AddCommand(cmd.Build(&downloadOperation{}))
	c.AddCommand(cmd.Build(&listOperation{}))
	c.AddCommand(cmd.Build(&mkdirOperation{}))
	c.AddCommand(cmd.Build(&removeOperation{}))
	c.AddCommand(cmd.Build(&statOperation{}))
	c.AddCommand(cmd.Build(&renameOperation{}))
	return c
}

func open(ctx cmd.OperationContext) (*volumecontent.Content, error) {
	id, err := uuid.Parse(ctx.Args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid volume ID %q: expected UUID: %w", ctx.Args[0], err)
	}
	for _, value := range ctx.Args[1:] {
		if value == "" || strings.HasPrefix(value, "/") || strings.ContainsRune(value, 0) {
			return nil, fmt.Errorf("invalid volume path %q: use a relative path; . denotes the root", value)
		}
		for _, part := range strings.Split(value, "/") {
			if part == ".." {
				return nil, fmt.Errorf("invalid volume path %q: .. components are not allowed", value)
			}
		}
	}
	return volumecontent.NewWithTransport(id, ctx.Client.Transport())
}
