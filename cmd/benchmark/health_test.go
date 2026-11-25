package benchmark

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
)

func parseHealth(t *testing.T, args ...string) *healthOperation {
	t.Helper()
	op := &healthOperation{}
	require.NoError(t, op.Command().ParseFlags(args))
	return op
}

func TestHealthDefaults(t *testing.T) {
	op := parseHealth(t)

	assert.Equal(t, 1, op.count, "one sandbox by default")
	assert.Equal(t, 1, op.concurrency)
	assert.Equal(t, []string{"https://www.apple.com/library/test/success.html"}, op.urls)
	assert.Empty(t, op.intranetHost, "resolved from the region")
	assert.Equal(t, 8000, op.publicURLPort)
	assert.NoError(t, op.validate())
}

func TestHealthValidate(t *testing.T) {
	tests := []struct {
		name string
		args []string
		err  string
	}{
		{"url without scheme", []string{"--url", "www.baidu.com"}, "--url"},
		{"ftp url", []string{"--url", "ftp://example.com"}, "--url"},
		{"port too low", []string{"--public-url-port", "0"}, "--public-url-port"},
		{"port too high", []string{"--public-url-port", "70000"}, "--public-url-port"},
		{"zero count", []string{"-n", "0"}, "--count"},
		{"duplicate url", []string{"--url", "https://a.com", "--url", "https://a.com"}, "duplicate --url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseHealth(t, tt.args...).validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}

	op := parseHealth(t, "--url", "https://www.baidu.com", "--url", "http://example.com:8080/x", "-n", "20", "-c", "5")
	assert.Equal(t, []string{"https://www.baidu.com", "http://example.com:8080/x"}, op.urls)
	assert.NoError(t, op.validate())

	op = parseHealth(t, "--url", "https://example.com/api?ids=1,2")
	assert.Equal(t, []string{"https://example.com/api?ids=1,2"}, op.urls, "a comma does not split the URL")
	assert.NoError(t, op.validate())
}

func TestResolveIntranetHost(t *testing.T) {
	assert.Equal(t, "internal.s3-cn-wlcb.ufileos.com", resolveIntranetHost("cn-wlcb", ""))
	assert.Equal(t, "internal.s3-cn-sh2.ufileos.com", resolveIntranetHost("cn-sh", ""))
	assert.Equal(t, "internal.s3-us-ca.ufileos.com", resolveIntranetHost("us-ca", ""))
	assert.Equal(t, "internal.s3-us-den.ufileos.com", resolveIntranetHost("us-den", ""))
	assert.Equal(t, "internal.s3-vn-sng.ufileos.com", resolveIntranetHost("vn-sng", ""))

	assert.Equal(t, "internal.s3-mars-1.ufileos.com", resolveIntranetHost("mars-1", ""),
		"a new region follows the naming pattern")
	assert.Equal(t, "10.0.0.1", resolveIntranetHost("cn-wlcb", "10.0.0.1"), "the flag wins")
	assert.Equal(t, "10.0.0.1", resolveIntranetHost("mars-1", "10.0.0.1"))
}

func TestHealthSteps(t *testing.T) {
	phases := func(steps []benchmark.Step[*benchSandbox]) []benchmark.Phase {
		var out []benchmark.Phase
		for _, s := range steps {
			out = append(out, s.Phase)
		}
		return out
	}

	op := parseHealth(t, "--url", "https://www.baidu.com", "--url", "https://example.com/path")
	assert.Equal(t, []benchmark.Phase{
		"exec",
		"filesystem",
		"internet https://www.baidu.com",
		"internet https://example.com/path",
		"intranet internal.s3-cn-wlcb.ufileos.com",
		"public-url 8000",
		"pause",
		"resume",
	}, phases(op.steps(nil, "internal.s3-cn-wlcb.ufileos.com")))
}

func TestProbeCommand(t *testing.T) {
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 -w '%{http_code}' 'https://www.baidu.com'`,
		probeCommand("https://www.baidu.com"))
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 -w '%{http_code}' 'http://x/'"'"'; rm -rf /'`,
		probeCommand("http://x/'; rm -rf /"), "a quote in the URL cannot break out")
}

func TestPingCommand(t *testing.T) {
	assert.Equal(t, "sudo -n ping -c 3 -W 2 'internal.s3-cn-wlcb.ufileos.com'", pingCommand("internal.s3-cn-wlcb.ufileos.com"))
	assert.Equal(t, `sudo -n ping -c 3 -W 2 'x'"'"'; rm -rf /'`, pingCommand("x'; rm -rf /"),
		"a quote in the host cannot break out")
}

func TestPingSummary(t *testing.T) {
	out := `PING internal-cn-wlcb.ufileos.com (10.0.0.1) 56(84) bytes of data.

--- internal-cn-wlcb.ufileos.com ping statistics ---
3 packets transmitted, 0 received, 100% packet loss, time 2034ms
`
	assert.Equal(t, "3 packets transmitted, 0 received, 100% packet loss, time 2034ms", pingSummary(out))
	assert.Empty(t, pingSummary("ping: unknown host"))
	assert.Empty(t, pingSummary(""))
}

func TestCheckHTTPCode(t *testing.T) {
	assert.NoError(t, checkHTTPCode("200"))
	assert.NoError(t, checkHTTPCode("200\n"))

	for _, code := range []string{"301", "403", "502"} {
		err := checkHTTPCode(code)
		require.Error(t, err, code)
		assert.Equal(t, "HTTP "+code+", want 200", err.Error())
	}

	for _, out := range []string{"000", ""} {
		err := checkHTTPCode(out)
		require.Error(t, err, out)
		assert.Equal(t, "no HTTP response", err.Error())
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			if r.Header.Get("E2B-Traffic-Access-Token") != "tok" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			_, _ = w.Write([]byte("token-123"))
		default:
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	defer srv.Close()

	body, err := fetch(context.Background(), srv.Client(), srv.URL+"/health", "tok")
	require.NoError(t, err)
	assert.Equal(t, "token-123", body)

	_, err = fetch(context.Background(), srv.Client(), srv.URL+"/health", "")
	require.Error(t, err, "the traffic token is sent when given")
	assert.Contains(t, err.Error(), "HTTP 403")

	_, err = fetch(context.Background(), srv.Client(), srv.URL+"/starting", "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 502")
}

func TestRunViewHideBreakdown(t *testing.T) {
	v := newRunView("run-1", "system/base", 1, 300, sampleReport())
	v.hideBreakdown = true

	var b strings.Builder
	require.NoError(t, v.writePretty(&b))
	out := b.String()

	assert.Contains(t, out, "| create ")
	assert.NotContains(t, out, "connect |")
	assert.NotContains(t, out, "service |")
}

func healthPhases(t *testing.T, args ...string) []benchmark.Phase {
	t.Helper()
	op := parseHealth(t, args...)
	require.NoError(t, op.validate())
	var out []benchmark.Phase
	for _, s := range op.steps(nil, "internal.s3-cn-wlcb.ufileos.com") {
		out = append(out, s.Phase)
	}
	return out
}

func TestHealthSelectChecks(t *testing.T) {
	assert.Equal(t, []benchmark.Phase{"internet " + defaultInternetURL, "intranet internal.s3-cn-wlcb.ufileos.com"},
		healthPhases(t, "--only", "internet,intranet"))

	assert.Equal(t, []benchmark.Phase{"exec", "intranet internal.s3-cn-wlcb.ufileos.com"},
		healthPhases(t, "--only", "intranet", "--only", "exec"),
		"checks run in their fixed order, whatever the order given")

	assert.Equal(t, []benchmark.Phase{"pause", "resume"}, healthPhases(t, "--only", "pause"),
		"pause covers resume")

	assert.Equal(t, []benchmark.Phase{
		"exec", "filesystem", "internet " + defaultInternetURL, "intranet internal.s3-cn-wlcb.ufileos.com",
	}, healthPhases(t, "--skip", "pause,public-url"))
}

func TestHealthEveryCheckHasSteps(t *testing.T) {
	for _, check := range healthChecks {
		assert.NotEmpty(t, healthPhases(t, "--only", check), check)
	}
}

func TestHealthValidateChecks(t *testing.T) {
	err := parseHealth(t, "--skip", "pause", "--only", "exec").validate()
	require.Error(t, err)
	assert.Equal(t, "--skip and --only cannot be used together", err.Error())

	err = parseHealth(t, "--skip", "resume").validate()
	require.Error(t, err)
	assert.Equal(t, `invalid --skip "resume": must be one of exec, filesystem, internet, intranet, public-url, pause`, err.Error())

	err = parseHealth(t, "--only", "dns").validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), `invalid --only "dns"`)
}

func TestHealthPublicURLNamesItsPort(t *testing.T) {
	assert.Equal(t, []benchmark.Phase{"public-url 3000"},
		healthPhases(t, "--only", "public-url", "--public-url-port", "3000"))
}
