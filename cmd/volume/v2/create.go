package volume

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2/api"
)

type createOperation struct {
	req         api.CreateVolume
	params      api.CreateVolumeParams
	extraParams *string
}

func (o *createOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"cr"},
		Short:   "Create a v2 volume",
		Args:    cobra.ExactArgs(1),
	}
	c.Flags().Int64Var(&o.req.SizeGiB, "size-gib", 0,
		"Capacity in GiB (Basic requires at least 500; Advanced requires at least 100)")
	c.MarkFlagRequired("size-gib")
	flags.NullableStringVarP(c.Flags(), &o.extraParams, "extra-params", "",
		`Provider options as a JSON object, e.g. {"storageType":"Advanced"}`)
	flags.NullableStringVarP(c.Flags(), &o.params.IdempotencyKey, "idempotency-key", "",
		"Key for safely replaying the same create request")
	return c
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	if err := validateSize(o.req.SizeGiB); err != nil {
		return err
	}
	req := o.req
	req.Name = strings.TrimSpace(ctx.Args[0])
	if req.Name == "" {
		return fmt.Errorf("volume name cannot be empty")
	}
	if o.extraParams != nil {
		var extra api.ExtraParams
		if err := json.Unmarshal([]byte(*o.extraParams), &extra); err != nil {
			return fmt.Errorf("invalid --extra-params: expected a JSON object: %w", err)
		}
		if extra == nil {
			return fmt.Errorf("invalid --extra-params: expected a JSON object")
		}
		req.ExtraParams = &extra
	}
	vol, err := ctx.Client.VolumesV2().Create(ctx, req, &o.params)
	if err != nil {
		return err
	}
	fmt.Printf("Volume created: %s\n", vol.VolumeID)
	return nil
}
