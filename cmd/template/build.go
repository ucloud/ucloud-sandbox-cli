package template

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
)

const (
	fallbackDockerfileName = "Dockerfile"
	defaultDockerfileName  = "template.dockerfile"
)

type buildFlags struct {
	path             string
	dockerfile       string
	startCmd         string
	readyCmd         string
	cpuCount         int
	memoryMB         int
	cpuSet           bool
	memorySet        bool
	noCache          bool
	publish          bool
	tags             []string
	logLevel         string
	registryUsername string
	registryPassword string
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
		Use:     use + " [template-name]",
		Aliases: aliases,
		Short:   short,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.cpuSet = cmd.Flags().Changed("cpu-count")
			flags.memorySet = cmd.Flags().Changed("memory-mb")
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return runBuild(name, &flags)
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
	cmd.Flags().StringVar(&flags.logLevel, "level", "info", "Minimum build log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&flags.registryUsername, "registry-username", "", "Username for pulling the base image from a private registry")
	cmd.Flags().StringVar(&flags.registryPassword, "registry-password", "", "Password for pulling the base image from a private registry")
	return cmd
}

func runBuild(name string, flags *buildFlags) error {
	contextPath, localCfg := resolveBuildContext(name, flags.path)
	if name == "" && localCfg != nil {
		name = localCfg.TemplateName
	}
	if name == "" {
		return fmt.Errorf("template name is required; provide it as an argument or in ucloud-template.json")
	}
	if err := template.ValidateName(name); err != nil {
		return err
	}
	if localCfg != nil {
		if !flags.cpuSet && localCfg.CPUCount > 0 {
			flags.cpuCount = localCfg.CPUCount
		}
		if !flags.memorySet && localCfg.MemoryMB > 0 {
			flags.memoryMB = localCfg.MemoryMB
		}
		if flags.dockerfile == "" && localCfg.Dockerfile != "" {
			flags.dockerfile = localCfg.Dockerfile
		}
	}
	if flags.cpuCount <= 0 {
		return fmt.Errorf("CPU count must be greater than zero, got %d", flags.cpuCount)
	}
	if flags.memoryMB <= 0 {
		return fmt.Errorf("memory must be greater than zero, got %d", flags.memoryMB)
	}
	if flags.memoryMB%2 != 0 {
		return fmt.Errorf("memory must be an even number, got %d", flags.memoryMB)
	}
	if flags.startCmd == "" && flags.readyCmd != "" {
		return fmt.Errorf("both --cmd and --ready-cmd must be provided together")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	applyRegistryCredentials(flags, cfg)
	if (flags.registryUsername == "") != (flags.registryPassword == "") {
		return fmt.Errorf("both --registry-username and --registry-password must be provided together (or set them in the config file)")
	}

	client, err := config.NewClient(cfg)
	if err != nil {
		return err
	}
	dockerfilePath, err := resolveDockerfilePath(contextPath, flags.dockerfile)
	if err != nil {
		return err
	}
	builder, err := client.Templates().NewBuilder(template.BuilderOptions{
		FileContextPath: contextPath,
	}).FromDockerfile(dockerfilePath)
	if err != nil {
		return err
	}
	if flags.startCmd != "" {
		builder.SetStartCmd(flags.startCmd, template.ReadyCmd{Cmd: flags.readyCmd})
	}
	opts := template.BuildOptions{
		CPUCount:  flags.cpuCount,
		MemoryMB:  flags.memoryMB,
		SkipCache: flags.noCache,
	}
	if len(flags.tags) > 0 {
		opts.Tags = flags.tags
	}
	if flags.publish {
		opts.Publish = flags.publish
	}
	if flags.registryUsername != "" {
		opts.Registry = template.BasicAuth(flags.registryUsername, flags.registryPassword)
	}
	fmt.Println("\nBuilding sandbox template...")
	fmt.Println()
	info, err := client.Templates().Build(context.Background(), builder, name, opts)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	fmt.Printf("\n✅ Building sandbox template finished.\n")
	fmt.Printf("Template ID: %s\n", info.TemplateID)
	fmt.Printf("Build ID: %s\n", info.BuildID)
	fmt.Printf("\nYou can now use the template to create sandboxes.\n")
	return nil
}

// applyRegistryCredentials fills missing registry credentials from the global
// config; explicit flags take precedence.
func applyRegistryCredentials(flags *buildFlags, cfg *config.Config) {
	if flags.registryUsername == "" {
		flags.registryUsername = cfg.RegistryUsername
	}
	if flags.registryPassword == "" {
		flags.registryPassword = cfg.RegistryPassword
	}
}

// resolveBuildContext prefers the template's named directory, then --path itself.
func resolveBuildContext(name, path string) (string, *LocalConfig) {
	candidate := filepath.Join(path, name)
	if cfg, err := loadConfig(candidate); err == nil {
		return candidate, cfg
	}
	if cfg, err := loadConfig(path); err == nil {
		return path, cfg
	}
	return path, nil
}

// resolveDockerfilePath resolves the Dockerfile path from context or explicit input.
func resolveDockerfilePath(contextPath, explicit string) (string, error) {
	if explicit != "" {
		path := explicit
		if !filepath.IsAbs(path) {
			path = filepath.Join(contextPath, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("dockerfile %q: %w", path, err)
		}
		return path, nil
	}
	if cfg, err := loadConfig(contextPath); err == nil && cfg.Dockerfile != "" {
		path := cfg.Dockerfile
		if !filepath.IsAbs(path) {
			path = filepath.Join(contextPath, path)
		}
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("dockerfile %q: %w", path, err)
		}
		return path, nil
	}
	for _, filename := range []string{defaultDockerfileName, fallbackDockerfileName} {
		path := filepath.Join(contextPath, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no Dockerfile found in %q", contextPath)
}
