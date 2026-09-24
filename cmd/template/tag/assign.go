package tag

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

type assignOperation struct{}

func (o *assignOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:     "assign <template[:tag]> <tag> [tag...]",
		Aliases: []string{"add"},
		Short:   "Point tags at a template's build",
		// The target names the build, and there is nothing to do without at
		// least one tag to point at it.
		Args: cobra.MinimumNArgs(2),
	}
}

func (o *assignOperation) Run(ctx cmd.OperationContext) error {
	target, tags := ctx.Args[0], ctx.Args[1:]

	assigned, err := ctx.Client.Templates().AssignTags(ctx, api.AssignTemplateTagsRequest{
		Target: target,
		Tags:   tags,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Assigned tags %s to build %s.\n", strings.Join(assigned.Tags, ", "), assigned.BuildID)

	return nil
}
