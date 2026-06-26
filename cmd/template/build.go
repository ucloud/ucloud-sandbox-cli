package template

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/template"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

type buildFlags struct {
	path       string
	dockerfile string
	startCmd   string
	readyCmd   string
	cpuCount   int
	memoryMB   int
	noCache    bool
	publish    bool
	tags       []string
}

func newBuildCmd() *cobra.Command {
	return buildCommand("build", []string{"bd"}, "Build template from Dockerfile")
}

func newCreateCmd() *cobra.Command {
	return buildCommand("create", []string{"ct"}, "Create template from Dockerfile")
}

func buildCommand(use string, aliases []string, short string) *cobra.Command {
	var flags buildFlags

	cmd := &cobra.Command{
		Use:     use + " <template-name>",
		Aliases: aliases,
		Short:   short,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(args[0], &flags)
		},
	}

	cmd.Flags().StringVarP(&flags.path, "path", "p", ".", "Build context path")
	cmd.Flags().StringVarP(&flags.dockerfile, "dockerfile", "d", "", "Dockerfile path")
	cmd.Flags().StringVar(&flags.startCmd, "cmd", "", "Start command")
	cmd.Flags().StringVar(&flags.readyCmd, "ready-cmd", "", "Ready probe command")
	cmd.Flags().IntVar(&flags.cpuCount, "cpu-count", 2, "CPU count")
	cmd.Flags().IntVar(&flags.memoryMB, "memory-mb", 1024, "Memory in MB (must be even)")
	cmd.Flags().BoolVar(&flags.noCache, "no-cache", false, "Skip build cache")
	cmd.Flags().BoolVar(&flags.publish, "publish", false, "Publish after build")
	cmd.Flags().StringSliceVarP(&flags.tags, "tag", "t", nil, "Build tags")

	return cmd
}

func runBuild(name string, flags *buildFlags) error {
	// 1. Validate name
	if err := template.ValidateName(name); err != nil {
		return err
	}

	// 2. Validate memory (must be even)
	if flags.memoryMB%2 != 0 {
		return fmt.Errorf("memory must be an even number, got %d", flags.memoryMB)
	}

	// 3. Validate start/ready cmd
	if flags.startCmd == "" && flags.readyCmd != "" {
		return fmt.Errorf("both --cmd and --ready-cmd must be provided together")
	}

	// 4. Load config and create client
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	client, err := config.NewClient(cfg)
	if err != nil {
		return err
	}

	// 5. Build template
	ctx := context.Background()

	// Create builder (simplified: FromBaseImage + RunCmd)
	builder := sdk.NewTemplate(sdk.WithFileContextPath(flags.path)).FromBaseImage()

	if flags.startCmd != "" {
		builder = builder.SetStartCmd(flags.startCmd, sdk.ReadyCmd{Cmd: flags.readyCmd})
	}

	// Build options
	opts := []sdk.BuildOption{
		sdk.WithBuildCPUCount(flags.cpuCount),
		sdk.WithBuildMemoryMB(flags.memoryMB),
		sdk.WithBuildSkipCache(flags.noCache),
		sdk.WithOnBuildLogs(sdk.DefaultBuildLogger()),
	}
	if len(flags.tags) > 0 {
		opts = append(opts, sdk.WithBuildTags(flags.tags))
	}
	if flags.publish {
		opts = append(opts, sdk.WithPublishTemplate())
	}

	fmt.Println("\nBuilding sandbox template...")
	fmt.Println()

	info, err := client.BuildTemplate(ctx, builder, name, opts...)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// 6. Print success
	fmt.Printf("\n✅ Building sandbox template finished.\n")
	fmt.Printf("Template ID: %s\n", info.TemplateID)
	fmt.Printf("Build ID: %s\n", info.BuildID)
	fmt.Printf("\nYou can now use the template to create sandboxes.\n")

	return nil
}
