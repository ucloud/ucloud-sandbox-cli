package sandbox

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

func newCreateCmd() *cobra.Command {
	var timeout int
	var detach bool
	var autoPause bool
	var autoResume bool
	var mountSpecs []string

	cmd := &cobra.Command{
		Use:     "create [template]",
		Aliases: []string{"cr"},
		Short:   "Create a new sandbox",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			template := "system/base"
			if len(args) > 0 {
				template = args[0]
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
			opts := []sdk.SandboxOption{sdk.WithTemplate(template)}
			if timeout > 0 {
				opts = append(opts, sdk.WithTimeout(timeout))
			}
			if autoPause {
				opts = append(opts, sdk.WithAutoPause(true))
			}
			if autoResume {
				opts = append(opts, sdk.WithAutoResume(sdk.AutoResumePolicyOn))
			}
			if len(mountSpecs) > 0 {
				mounts, err := parseVolumeMounts(mountSpecs)
				if err != nil {
					return err
				}
				opts = append(opts, sdk.WithVolumeMounts(mounts))
			}

			sbx, err := client.CreateSandbox(ctx, opts...)
			if err != nil {
				return err
			}

			fmt.Printf("Sandbox created: %s (template: %s)\n", sbx.ID, template)

			if detach {
				return nil
			}
			return connectTerminal(ctx, sbx)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 0, "Sandbox timeout in seconds")
	cmd.Flags().BoolVar(&detach, "detach", false, "Do not connect to the sandbox after creation")
	cmd.Flags().BoolVar(&autoPause, "auto-pause", false, "Automatically pause the sandbox when the timeout is reached")
	cmd.Flags().BoolVar(&autoResume, "auto-resume", false, "Automatically resume the sandbox when it is paused")
	cmd.Flags().StringArrayVar(&mountSpecs, "mount", nil, "Mount volume as <volume-name>:<path> (repeatable)")
	return cmd
}

func parseVolumeMounts(values []string) ([]sdk.VolumeMount, error) {
	mounts := make([]sdk.VolumeMount, 0, len(values))
	for _, value := range values {
		volumeName, mountPath, ok := strings.Cut(value, ":")
		if !ok || volumeName == "" || mountPath == "" {
			return nil, fmt.Errorf("invalid mount %q: expected <volume-name>:<absolute-path>", value)
		}
		if !strings.HasPrefix(mountPath, "/") {
			return nil, fmt.Errorf("invalid mount %q: mount path must be absolute", value)
		}
		mounts = append(mounts, sdk.VolumeMount{Name: volumeName, Path: mountPath})
	}
	return mounts, nil
}
