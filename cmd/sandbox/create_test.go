package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// parseCreateFlags registers the create flags the way the command does and
// returns the request they filled in.
func parseCreateFlags(t *testing.T, args ...string) *api.NewSandbox {
	t.Helper()

	op := &createOperation{}
	require.NoError(t, op.Command().ParseFlags(args))

	return &op.req
}

func parseCreateFlagsErr(t *testing.T, args ...string) error {
	t.Helper()

	op := &createOperation{}

	return op.Command().ParseFlags(args)
}

func TestCreateFlagsStayNilWhenNotProvided(t *testing.T) {
	req := parseCreateFlags(t)

	assert.Nil(t, req.AllowInternetAccess)
	assert.Nil(t, req.AutoPause)
	assert.Nil(t, req.AutoPauseMemory)
	assert.Nil(t, req.AutoResume)
	assert.Nil(t, req.EnvVars)
	assert.Nil(t, req.Metadata)
	assert.Nil(t, req.Network)
	assert.Nil(t, req.Secure)
	assert.Nil(t, req.Timeout)
	assert.Nil(t, req.VolumeMounts)
}

func TestCreateAutoResumeFlag(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		enabled bool
	}{
		{args: []string{"--auto-resume"}, enabled: true},
		{args: []string{"--auto-resume=true"}, enabled: true},
		{args: []string{"--auto-resume=false"}, enabled: false},
	} {
		req := parseCreateFlags(t, tc.args...)

		require.NotNil(t, req.AutoResume, "args %v", tc.args)
		assert.Equal(t, tc.enabled, req.AutoResume.Enabled, "args %v", tc.args)
		assert.Nil(t, req.Network, "auto-resume must not allocate the network config")
	}
}

func TestCreateSimpleFlags(t *testing.T) {
	req := parseCreateFlags(t,
		"--allow-internet-access=false",
		"--auto-pause",
		"--auto-pause-memory=false",
		"-e", "FOO: bar",
		"--env", "BAZ:qux",
		"-m", "user: abc",
		"--secure",
		"--timeout", "300",
	)

	require.NotNil(t, req.AllowInternetAccess)
	assert.False(t, *req.AllowInternetAccess)
	require.NotNil(t, req.AutoPause)
	assert.True(t, *req.AutoPause)
	require.NotNil(t, req.AutoPauseMemory)
	assert.False(t, *req.AutoPauseMemory)
	require.NotNil(t, req.EnvVars)
	assert.Equal(t, api.EnvVars{"FOO": "bar", "BAZ": "qux"}, *req.EnvVars)
	require.NotNil(t, req.Metadata)
	assert.Equal(t, api.SandboxMetadata{"user": "abc"}, *req.Metadata)
	require.NotNil(t, req.Secure)
	assert.True(t, *req.Secure)
	require.NotNil(t, req.Timeout)
	assert.Equal(t, int32(300), *req.Timeout)
}

func TestCreateNetworkFlags(t *testing.T) {
	req := parseCreateFlags(t,
		"--network-allow-out", "8.8.8.8/32",
		"--network-allow-out", "*.example.com",
		"--network-allow-public-traffic",
		"--network-deny-out", "10.0.0.0/8",
		"--network-egress-proxy-address", "proxy.example.com:1080",
		"--network-egress-proxy-username", "user",
		"--network-egress-proxy-password", "pass",
		"--network-https-ports", "443",
		"--network-https-ports", "8443",
		"--network-mask-request-host", "masked.example.com",
	)

	require.NotNil(t, req.Network)
	require.NotNil(t, req.Network.AllowOut)
	assert.Equal(t, []string{"8.8.8.8/32", "*.example.com"}, *req.Network.AllowOut)
	require.NotNil(t, req.Network.AllowPublicTraffic)
	assert.True(t, *req.Network.AllowPublicTraffic)
	require.NotNil(t, req.Network.DenyOut)
	assert.Equal(t, []string{"10.0.0.0/8"}, *req.Network.DenyOut)
	require.NotNil(t, req.Network.EgressProxy)
	assert.Equal(t, "proxy.example.com:1080", req.Network.EgressProxy.Address)
	require.NotNil(t, req.Network.EgressProxy.Username)
	assert.Equal(t, "user", *req.Network.EgressProxy.Username)
	require.NotNil(t, req.Network.EgressProxy.Password)
	assert.Equal(t, "pass", *req.Network.EgressProxy.Password)
	require.NotNil(t, req.Network.HttpsPorts)
	assert.Equal(t, []uint32{443, 8443}, *req.Network.HttpsPorts)
	require.NotNil(t, req.Network.MaskRequestHost)
	assert.Equal(t, "masked.example.com", *req.Network.MaskRequestHost)
	assert.Nil(t, req.Network.Rules)

	assert.Nil(t, req.AutoResume, "network flags must not allocate the auto-resume config")
}

func TestCreateNetworkConfigAllocatesOnlyWhatIsProvided(t *testing.T) {
	req := parseCreateFlags(t, "--network-allow-public-traffic=false")

	require.NotNil(t, req.Network)
	require.NotNil(t, req.Network.AllowPublicTraffic)
	assert.False(t, *req.Network.AllowPublicTraffic)

	// Every other field, including the nested egress proxy, stays nil.
	assert.Nil(t, req.Network.AllowOut)
	assert.Nil(t, req.Network.DenyOut)
	assert.Nil(t, req.Network.EgressProxy)
	assert.Nil(t, req.Network.HttpsPorts)
	assert.Nil(t, req.Network.MaskRequestHost)
	assert.Nil(t, req.Network.Rules)
}

func TestCreateEgressProxyAllocatesOnASingleFlag(t *testing.T) {
	req := parseCreateFlags(t, "--network-egress-proxy-username", "user")

	require.NotNil(t, req.Network)
	require.NotNil(t, req.Network.EgressProxy)
	require.NotNil(t, req.Network.EgressProxy.Username)
	assert.Equal(t, "user", *req.Network.EgressProxy.Username)
	assert.Empty(t, req.Network.EgressProxy.Address)
	assert.Nil(t, req.Network.EgressProxy.Password)
}

func TestCreateNetworkRules(t *testing.T) {
	req := parseCreateFlags(t, "--network-rules",
		`{"mydomain.com": {"X-API-KEY": "${api_key}", "X-Sandbox-ID": "${sandbox_id}"}}`)

	require.NotNil(t, req.Network)
	require.NotNil(t, req.Network.Rules)

	rules := (*req.Network.Rules)["mydomain.com"]
	require.Len(t, rules, 1)
	require.NotNil(t, rules[0].Transform)
	require.NotNil(t, rules[0].Transform.Headers)

	// The e2b.secrets prefix is added on the user's behalf.
	assert.Equal(t, map[string]string{
		"X-API-KEY":    "${e2b.secrets.api_key}",
		"X-Sandbox-ID": "${e2b.secrets.sandbox_id}",
	}, *rules[0].Transform.Headers)
}

func TestCreateNetworkRulesHeaderValues(t *testing.T) {
	req := parseCreateFlags(t, "--network-rules", `{"mydomain.com": {
		"X-Plain": "application/json",
		"Authorization": "Bearer ${api_key}",
		"X-Already-Qualified": "${e2b.secrets.api_key}",
		"X-Two": "${a}/${b}"
	}}`)

	require.NotNil(t, req.Network.Rules)
	rules := (*req.Network.Rules)["mydomain.com"]
	require.Len(t, rules, 1)

	assert.Equal(t, map[string]string{
		"X-Plain":             "application/json",
		"Authorization":       "Bearer ${e2b.secrets.api_key}",
		"X-Already-Qualified": "${e2b.secrets.api_key}",
		"X-Two":               "${e2b.secrets.a}/${e2b.secrets.b}",
	}, *rules[0].Transform.Headers)
}

func TestCreateNetworkRulesRepeatedFlag(t *testing.T) {
	req := parseCreateFlags(t,
		"--network-rules", `{"a.com": {"X-A": "1"}, "b.com": {"X-B": "2"}}`,
		"--network-rules", `{"a.com": {"X-A2": "3"}}`,
	)

	require.NotNil(t, req.Network.Rules)
	rules := *req.Network.Rules

	// The second occurrence replaces the domain it mentions, and leaves the
	// other one alone.
	require.Len(t, rules, 2)
	require.Len(t, rules["a.com"], 1)
	assert.Equal(t, map[string]string{"X-A2": "3"}, *rules["a.com"][0].Transform.Headers)
	require.Len(t, rules["b.com"], 1)
	assert.Equal(t, map[string]string{"X-B": "2"}, *rules["b.com"][0].Transform.Headers)
}

func TestCreateNetworkRulesRejectsInvalidValue(t *testing.T) {
	for _, rules := range []string{
		`not json`,
		`{"a.com": {"X-A": 1}}`,        // a header value must be a string
		`{"": {"X-A": "1"}}`,           // the domain cannot be empty
		`{"a.com": {}}`,                // at least one header is required
		`{"a.com": {"X-A": "${}"}}`,    // an empty secret name
		`{"a.com": {"X-A": "${a b}"}}`, // an invalid secret name
	} {
		err := parseCreateFlagsErr(t, "--network-rules", rules)
		assert.Error(t, err, "rules %s should be rejected", rules)
	}
}

func TestCreateMountFlag(t *testing.T) {
	req := parseCreateFlags(t,
		"--mount", "data:/mnt/data",
		"--mount", " cache : /mnt/cache ",
	)

	require.NotNil(t, req.VolumeMounts)
	assert.Equal(t, []api.SandboxVolumeMount{
		{Name: "data", Path: "/mnt/data"},
		{Name: "cache", Path: "/mnt/cache"},
	}, *req.VolumeMounts)
}

func TestCreateMountFlagRejectsInvalidValue(t *testing.T) {
	for _, mount := range []string{"data", "", ":", "data:", ":/mnt/data"} {
		err := parseCreateFlagsErr(t, "--mount", mount)
		assert.Error(t, err, "mount %q should be rejected", mount)
	}
}
