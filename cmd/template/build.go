package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	"github.com/ucloud/ucloud-sandbox-cli/internal/registry"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template/build"
)

// The file names `build` looks for when none is given, in order.
const (
	defaultDockerfileName  = "template.dockerfile"
	fallbackDockerfileName = "Dockerfile"
)

type buildOperation struct {
	req api.TemplateBuildRequestV3

	path       string
	dockerfile string

	startCmd string
	readyCmd string

	noCache bool
	publish bool

	logLevel string
}

func (o *buildOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:     "build [template-name]",
		Aliases: []string{"bd", "create", "ct"},
		Short:   "Build a template from a Dockerfile",
		Args:    cobra.MaximumNArgs(1),
	}

	c.Flags().StringVarP(&o.path, "path", "p", ".", "Project root the template directory is looked for in")
	c.Flags().StringVarP(&o.dockerfile, "dockerfile", "d", "",
		"Dockerfile to build, relative to the template directory")

	c.Flags().StringVarP(&o.startCmd, "cmd", "", "", "Command the sandbox starts with")
	c.Flags().StringVarP(&o.readyCmd, "ready-cmd", "", "",
		"Command that decides when the started sandbox is ready (needs --cmd)")

	flags.NullableInt32VarP(c.Flags(), &o.req.CpuCount, "cpu-count", "",
		"CPU cores of sandboxes built from this template")
	flags.NullableInt32VarP(c.Flags(), &o.req.MemoryMB, "memory-mb", "",
		"Memory of sandboxes built from this template, in MiB (must be even)")
	flags.NullableInt32VarP(c.Flags(), &o.req.MinFreeDiskMb, "min-free-disk-mb", "",
		"Free disk space to leave after the build steps, in MiB")
	flags.NullableStringSliceVarP(c.Flags(), &o.req.Tags, "tag", "t",
		"Tag to assign to this build")

	c.Flags().BoolVarP(&o.noCache, "no-cache", "", false, "Rebuild every step, ignoring the cache")
	c.Flags().BoolVarP(&o.publish, "publish", "", false, "Publish the template once it is built")
	c.Flags().StringVarP(&o.logLevel, "level", "", string(api.LogLevelInfo),
		"Minimum build log level to print (debug, info, warn, error)")

	return c
}

func (o *buildOperation) Run(ctx cmd.OperationContext) error {
	var name string
	if len(ctx.Args) > 0 {
		name = ctx.Args[0]
	}

	contextPath, local := resolveBuildContext(name, o.path)

	req := o.req
	if name == "" && local != nil {
		name = local.TemplateName
	}
	if name == "" {
		return fmt.Errorf("template name is required: give it as an argument or in %s", configFileName)
	}
	if err := template.ValidateName(name); err != nil {
		return err
	}
	req.Name = &name

	// A flag that was not given falls back to the local config, and then to
	// whatever the platform defaults to.
	if local != nil {
		if req.CpuCount == nil && local.CPUCount > 0 {
			req.CpuCount = &local.CPUCount
		}
		if req.MemoryMB == nil && local.MemoryMB > 0 {
			req.MemoryMB = &local.MemoryMB
		}
		if o.dockerfile == "" {
			o.dockerfile = local.Dockerfile
		}
	}

	if err := validateResources(req); err != nil {
		return err
	}
	if o.startCmd == "" && o.readyCmd != "" {
		return fmt.Errorf("--ready-cmd needs --cmd: there is nothing to wait for otherwise")
	}

	level := api.LogLevel(o.logLevel)
	if !level.Valid() {
		return fmt.Errorf("invalid --level %q: expected %s, %s, %s or %s",
			o.logLevel, api.LogLevelDebug, api.LogLevelInfo, api.LogLevelWarn, api.LogLevelError)
	}

	dockerfilePath, err := resolveDockerfilePath(contextPath, o.dockerfile)
	if err != nil {
		return err
	}

	// Parsing also fixes the file context to the Dockerfile's directory and
	// hashes every COPY source against it.
	builder, err := build.FromDockerfile(dockerfilePath)
	if err != nil {
		return err
	}

	if o.startCmd != "" {
		builder.SetStartCmd(o.startCmd)
		if o.readyCmd != "" {
			builder.SetReadyCmd(o.readyCmd)
		}
	}

	builder.Force(o.noCache)
	builder.SetLogger(build.DefaultLoggerWithLevel(level))

	if err := applyRegistryAuth(builder, ctx.Config); err != nil {
		return err
	}

	fmt.Println("\nBuilding sandbox template...")
	fmt.Println()

	info, err := builder.Build(ctx, ctx.Client, req)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	if o.publish {
		if err := ctx.Client.Templates().Publish(ctx, info.TemplateID); err != nil {
			return fmt.Errorf("built, but publishing failed: %w", err)
		}
	}

	fmt.Printf("\nBuilding sandbox template finished.\n")
	fmt.Printf("Template ID: %s\n", info.TemplateID)
	fmt.Printf("Build ID: %s\n", info.BuildID)

	// Recording the ID is what lets `delete` and `publish` find this template
	// without being told its ID again.
	rememberTemplateID(contextPath, local, name, info.TemplateID)

	fmt.Printf("\nYou can now use the template to create sandboxes.\n")

	return nil
}

// validateResources rejects the resource requests the platform would reject,
// but with an error that names the rule.
func validateResources(req api.TemplateBuildRequestV3) error {
	if req.CpuCount != nil && *req.CpuCount <= 0 {
		return fmt.Errorf("CPU count must be greater than zero, got %d", *req.CpuCount)
	}

	if req.MemoryMB != nil {
		if *req.MemoryMB <= 0 {
			return fmt.Errorf("memory must be greater than zero, got %d", *req.MemoryMB)
		}
		if *req.MemoryMB%2 != 0 {
			return fmt.Errorf("memory must be an even number, got %d", *req.MemoryMB)
		}
	}

	if req.MinFreeDiskMb != nil && *req.MinFreeDiskMb < 0 {
		return fmt.Errorf("minimum free disk must be zero or greater, got %d", *req.MinFreeDiskMb)
	}

	return nil
}

// applyRegistryAuth hands the builder the credentials configured for the
// registry the base image comes from, if there are any. A public image needs
// none, which is why a registry with nothing stored for it is not an error.
func applyRegistryAuth(builder *build.Builder, cfg *config.Config) error {
	image := builder.GetImage()
	if image == "" {
		return nil
	}

	domain, err := registry.Domain(image)
	if err != nil {
		return err
	}

	auth, ok := cfg.RegistryAuth(domain)
	if !ok {
		return nil
	}

	builder.SetImageRegistryAuth(auth.Username, auth.Password)

	return nil
}

// resolveBuildContext finds the directory holding the template, preferring one
// named after the template over the project root itself.
func resolveBuildContext(name, path string) (string, *LocalConfig) {
	if name != "" {
		candidate := filepath.Join(path, name)
		if cfg, err := loadConfig(candidate); err == nil {
			return candidate, cfg
		}
	}

	if cfg, err := loadConfig(path); err == nil {
		return path, cfg
	}

	return path, nil
}

// resolveDockerfilePath finds the Dockerfile to build: the one named, then the
// one the local config records, then the conventional names.
//
// The result must sit inside contextPath. The SDK fixes the build's file
// context to the Dockerfile's own directory and hashes COPY sources against it
// while parsing, so a Dockerfile elsewhere would upload one set of files under
// the hash of another.
func resolveDockerfilePath(contextPath, explicit string) (string, error) {
	if explicit == "" {
		if cfg, err := loadConfig(contextPath); err == nil {
			explicit = cfg.Dockerfile
		}
	}

	if explicit != "" {
		path := explicit
		if !filepath.IsAbs(path) {
			path = filepath.Join(contextPath, path)
		}
		if err := checkDockerfileDirectory(contextPath, path); err != nil {
			return "", err
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

// checkDockerfileDirectory rejects a Dockerfile that does not sit directly in
// the build context.
//
// The SDK fixes the build's file context to the Dockerfile's own directory and
// hashes every COPY source against it while parsing. A Dockerfile in a
// subdirectory would therefore resolve COPY relative to there rather than to
// the project root, and one outside would bundle files from somewhere else
// entirely -- both silently.
func checkDockerfileDirectory(contextPath, path string) error {
	context, err := filepath.Abs(contextPath)
	if err != nil {
		return err
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if filepath.Dir(absolute) != context {
		return fmt.Errorf(
			"dockerfile %q must sit in the build context %q: the build context is the Dockerfile's own directory",
			path, contextPath)
	}

	return nil
}

// rememberTemplateID writes the built template's ID into the local config so
// the other commands can find it. Failing to is reported but does not fail the
// build, which has already happened.
func rememberTemplateID(contextPath string, local *LocalConfig, name, templateID string) {
	if local == nil {
		return
	}

	if local.TemplateID == templateID {
		return
	}

	local.TemplateID = templateID
	if local.TemplateName == "" {
		local.TemplateName = name
	}

	if err := saveConfig(contextPath, local); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not record the template ID in %s: %v\n", configFileName, err)
	}
}
