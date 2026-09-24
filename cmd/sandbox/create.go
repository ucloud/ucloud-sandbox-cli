package sandbox

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/cmd/flags"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox/commands"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/secret"
)

type createOperation struct {
	req api.NewSandbox

	dryRun bool

	detached bool

	user string
}

func (o *createOperation) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create [template]",
		Aliases: []string{"cr"},
		Short:   "Create a sandbox",
		Args:    cobra.MaximumNArgs(1),
	}

	flags.NullableBoolVarP(cmd.Flags(), &o.req.AllowInternetAccess, "allow-internet-access", "",
		"Let the sandbox reach the internet; false denies egress to everything")
	flags.NullableBoolVarP(cmd.Flags(), &o.req.AutoPause, "auto-pause", "",
		"Pause the sandbox when it times out instead of shutting it down")
	flags.NullableBoolVarP(cmd.Flags(), &o.req.AutoPauseMemory, "auto-pause-memory", "",
		"Keep memory in the auto-pause snapshot; false cold-boots on resume and rules out --auto-resume (default true)")
	sandboxAutoResumeVarP(cmd.Flags(), &o.req.AutoResume, "auto-resume", "",
		"Let traffic resume the sandbox once it is paused")
	flags.NullableStringMapVarP(cmd.Flags(), &o.req.EnvVars, "env", "e",
		"Environment variable of the sandbox, as key: value")
	flags.NullableStringMapVarP(cmd.Flags(), &o.req.Metadata, "metadata", "m",
		"Metadata entry stored with the sandbox, as key: value")
	sandboxNetworkConfigVarP(cmd.Flags(), &o.req.Network, "network-")
	flags.NullableBoolVarP(cmd.Flags(), &o.req.Secure, "secure", "",
		"Secure all system communication with the sandbox")
	flags.NullableInt32VarP(cmd.Flags(), &o.req.Timeout, "timeout", "",
		"Seconds the sandbox lives for")
	cmd.Flags().VarP(&volumeMountsValue{target: &o.req.VolumeMounts}, "mount", "",
		"Volume to mount, as name:path")

	cmd.Flags().BoolVarP(&o.dryRun, "dry-run", "", false, "Show create request")

	cmd.Flags().BoolVarP(&o.detached, "detached", "", false, "Don't connect to sandbox terminal after creation")

	sandboxUserVarP(cmd.Flags(), &o.user)

	return cmd
}

// sandboxAutoResumeVarP registers the auto-resume flag. The config is
// allocated on first use, so the target stays nil when the flag is not
// provided.
func sandboxAutoResumeVarP(
	fs *pflag.FlagSet,
	target **api.SandboxAutoResumeConfig,
	name, shorthand, usage string,
) {
	flags.LazyBoolVarP(
		fs,
		target,
		func(config *api.SandboxAutoResumeConfig, enabled api.SandboxAutoResumeEnabled) {
			config.Enabled = enabled
		},
		name, shorthand, usage,
	)
}

// sandboxNetworkConfigVarP registers the network flags, each named after
// prefix (e.g. "network-"). The config is allocated on first use, so the
// target stays nil when none of the flags are provided.
func sandboxNetworkConfigVarP(
	fs *pflag.FlagSet,
	target **api.SandboxNetworkConfig,
	prefix string,
) {
	// The egress proxy is a nested config of its own, so it is allocated only
	// when one of its flags is provided.
	egressProxy := func(config *api.SandboxNetworkConfig) *api.SandboxEgressProxyConfig {
		if config.EgressProxy == nil {
			config.EgressProxy = &api.SandboxEgressProxyConfig{}
		}

		return config.EgressProxy
	}

	flags.LazyNullableStringSliceVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, allowOut *[]string) {
			config.AllowOut = allowOut
		},
		prefix+"allow-out", "",
		"Destination egress is allowed to, as a CIDR, an IP or a domain; allowing wins over denying",
	)

	flags.LazyNullableBoolVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, allowPublicTraffic *bool) {
			config.AllowPublicTraffic = allowPublicTraffic
		},
		prefix+"allow-public-traffic", "",
		"Serve the sandbox's URLs without requiring authentication",
	)

	flags.LazyNullableStringSliceVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, denyOut *[]string) {
			config.DenyOut = denyOut
		},
		prefix+"deny-out", "",
		"Destination egress is denied, as a CIDR or an IP; domains are not accepted here",
	)

	flags.LazyStringVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, address string) {
			egressProxy(config).Address = address
		},
		prefix+"egress-proxy-address", "",
		"SOCKS5 proxy the sandbox's outbound TCP is tunneled through, as host:port",
	)

	flags.LazyNullableStringVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, username *string) {
			egressProxy(config).Username = username
		},
		prefix+"egress-proxy-username", "",
		"Username for the egress proxy",
	)

	flags.LazyNullableStringVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, password *string) {
			egressProxy(config).Password = password
		},
		prefix+"egress-proxy-password", "",
		"Password for the egress proxy",
	)

	flags.LazyNullableUint32SliceVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, httpsPorts *[]uint32) {
			config.HttpsPorts = httpsPorts
		},
		prefix+"https-ports", "",
		"Sandbox port that serves HTTPS rather than plaintext HTTP",
	)

	flags.LazyNullableStringVarP(
		fs,
		target,
		func(config *api.SandboxNetworkConfig, maskRequestHost *string) {
			config.MaskRequestHost = maskRequestHost
		},
		prefix+"mask-request-host", "",
		"Host header sent with every request leaving the sandbox",
	)

	var rules *map[string][]api.SandboxNetworkRule

	flags.LazyVarP(
		fs,
		target,
		func(fs *pflag.FlagSet, name, shorthand, usage string) {
			fs.VarP(&networkRulesValue{target: &rules}, name, shorthand, usage)
		},
		func(config *api.SandboxNetworkConfig) {
			config.Rules = rules
		},
		prefix+"rules", "",
		`Headers to add to outgoing HTTPS requests, as {"domain": {"header": "value"}}`,
	)
}

// secretRefPrefix is what the platform expects in front of a secret name. It
// is added on the user's behalf, so a rule can reference "${api_key}".
const secretRefPrefix = "e2b.secrets."

// secretRefPattern matches a "${name}" reference inside a header value.
var secretRefPattern = regexp.MustCompile(`\$\{([^}]*)\}`)

// networkRulesValue binds a JSON flag holding the per-domain headers to the
// network transform rules:
//
//	{"mydomain.com": {"X-API-KEY": "${api_key}"}}
//
// Every occurrence replaces the rules of the domains it mentions and leaves
// the other domains alone.
type networkRulesValue struct {
	target **map[string][]api.SandboxNetworkRule
}

func (v *networkRulesValue) String() string {
	if *v.target == nil {
		return ""
	}

	encoded, err := json.Marshal(**v.target)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func (v *networkRulesValue) Set(s string) error {
	var domains map[string]map[string]string
	if err := json.Unmarshal([]byte(s), &domains); err != nil {
		return fmt.Errorf("invalid rules %q: expected {\"domain\": {\"header\": \"value\"}}: %w", s, err)
	}

	if *v.target == nil {
		rules := make(map[string][]api.SandboxNetworkRule, len(domains))
		*v.target = &rules
	}

	rules := **v.target

	for domain, headers := range domains {
		if domain == "" {
			return fmt.Errorf("invalid rules %q: domain cannot be empty", s)
		}

		if len(headers) == 0 {
			return fmt.Errorf("invalid rules %q: domain %q has no header", s, domain)
		}

		transformed := make(map[string]string, len(headers))

		for header, value := range headers {
			filled, err := fillSecretRefs(value)
			if err != nil {
				return fmt.Errorf("invalid rules %q: header %q of domain %q: %w", s, header, domain, err)
			}

			transformed[header] = filled
		}

		rules[domain] = []api.SandboxNetworkRule{{
			Transform: &api.SandboxNetworkTransform{Headers: &transformed},
		}}
	}

	return nil
}

func (v *networkRulesValue) Type() string {
	return "json"
}

// fillSecretRefs rewrites every "${name}" reference into the "${e2b.secrets.name}"
// form the platform resolves. A reference that already carries the prefix is
// left as it is, and anything outside a reference is kept verbatim.
func fillSecretRefs(value string) (string, error) {
	var err error

	filled := secretRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		if strings.HasPrefix(name, secretRefPrefix) {
			return match
		}

		ref, fillErr := secret.Fill(name)
		if fillErr != nil {
			if err == nil {
				err = fillErr
			}

			return match
		}

		return ref
	})

	if err != nil {
		return "", err
	}

	return filled, nil
}

// volumeMountsValue binds a repeatable "name:path" flag to the volume mounts.
// The slice is allocated on first use, so the target stays nil when the flag
// is not provided.
type volumeMountsValue struct {
	target **[]api.SandboxVolumeMount
}

func (v *volumeMountsValue) String() string {
	if *v.target == nil {
		return ""
	}

	parts := make([]string, 0, len(**v.target))
	for _, mount := range **v.target {
		parts = append(parts, mount.Name+":"+mount.Path)
	}

	return strings.Join(parts, ", ")
}

func (v *volumeMountsValue) Set(s string) error {
	name, path, ok := strings.Cut(s, ":")
	if !ok {
		return fmt.Errorf("invalid volume %q: expected name:path", s)
	}

	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)

	if name == "" || path == "" {
		return fmt.Errorf("invalid volume %q: neither the name nor the path can be empty", s)
	}

	if *v.target == nil {
		*v.target = &[]api.SandboxVolumeMount{}
	}

	mounts := *v.target
	*mounts = append(*mounts, api.SandboxVolumeMount{Name: name, Path: path})

	return nil
}

func (v *volumeMountsValue) Type() string {
	return "volumeMount"
}

func (o *createOperation) Run(ctx cmd.OperationContext) error {
	req := o.req
	var templateID string
	if len(ctx.Args) > 0 {
		templateID = ctx.Args[0]
	} else {
		templateID = "system/base"
	}
	req.TemplateID = templateID
	if o.dryRun {
		return cmd.ShowJSON(req)
	}
	sbx, err := ctx.Client.Sandboxes().Create(ctx, req, "")
	if err != nil {
		return err
	}

	fmt.Printf("Sandbox created: %s\n", sbx.SandboxID)

	if !o.detached {
		return connectTerminal(ctx, sbx, o.user, commands.Options{})
	}

	return nil
}
