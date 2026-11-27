package benchmark

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/internal/benchmark"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
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
	assert.Empty(t, op.intranetURL, "resolved from the region")
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
		{"intranet host without scheme", []string{"--intranet-url", "internal.s3-cn-sh2.ufileos.com"}, "--intranet-url"},
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

	assert.NoError(t, parseHealth(t, "--intranet-url", "http://10.0.0.1:8080/health").validate())

	op = parseHealth(t, "--url", "https://example.com/api?ids=1,2")
	assert.Equal(t, []string{"https://example.com/api?ids=1,2"}, op.urls, "a comma does not split the URL")
	assert.NoError(t, op.validate())
}

func TestIntranetURLFor(t *testing.T) {
	assert.Equal(t, "https://internal.s3-cn-wlcb.ufileos.com/", intranetURLFor("cn-wlcb"))
	assert.Equal(t, "https://internal.s3-cn-sh2.ufileos.com/", intranetURLFor("cn-sh"))
	assert.Equal(t, "https://internal.s3-us-ca.ufileos.com/", intranetURLFor("us-ca"))
	assert.Equal(t, "https://internal.s3-us-den.ufileos.com/", intranetURLFor("us-den"))
	assert.Equal(t, "https://internal.s3-vn-sng.ufileos.com/", intranetURLFor("vn-sng"))

	assert.Equal(t, "https://internal.s3-mars-1.ufileos.com/", intranetURLFor("mars-1"),
		"a new region follows the naming pattern")
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
		"intranet https://internal.s3-cn-wlcb.ufileos.com/",
		"public-url 8000",
		"pause",
		"resume",
	}, phases(op.steps(nil, "https://internal.s3-cn-wlcb.ufileos.com/")))
}

func TestProbeCommand(t *testing.T) {
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 -w '%{http_code}' 'https://www.baidu.com'`,
		probeCommand("https://www.baidu.com"))
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 -w '%{http_code}' 'http://x/'"'"'; rm -rf /'`,
		probeCommand("http://x/'; rm -rf /"), "a quote in the URL cannot break out")
}

func TestIntranetProbeCommand(t *testing.T) {
	const format = `-w '%{http_code} %{time_namelookup} %{time_connect} %{time_starttransfer}'`
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 `+format+` 'https://internal.s3-cn-sh2.ufileos.com/'`,
		intranetProbeCommand("https://internal.s3-cn-sh2.ufileos.com/"))
	assert.Equal(t,
		`curl -sS -o /dev/null -m 10 `+format+` 'http://x/'"'"'; rm -rf /'`,
		intranetProbeCommand("http://x/'; rm -rf /"), "a quote in the URL cannot break out")
}

func TestParseIntranetProbe(t *testing.T) {
	tests := []struct {
		name, stdout, code, timeline string
	}{
		{"answered", "400 0.178443 0.178742 0.184301", "400", "resolved 178ms, connected 179ms, first byte 184ms"},
		{"no connection", "000 5.012 0.000000 0.000000", "000", "resolved 5.012s, connected -, first byte -"},
		{"name did not resolve", "000 0.000000 0.000000 0.000000", "000", "resolved -, connected -, first byte -"},
		{"not curl's output", "curl: not found", "", ""},
		{"empty", "", "", ""},
		{"bad time", "200 x 0.1 0.2", "200", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, timeline := parseIntranetProbe(tt.stdout)
			assert.Equal(t, tt.code, code)
			assert.Equal(t, tt.timeline, timeline)
		})
	}
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

func healthPhases(t *testing.T, args ...string) []benchmark.Phase {
	t.Helper()
	op := parseHealth(t, args...)
	require.NoError(t, op.validate())
	var out []benchmark.Phase
	for _, s := range op.steps(nil, "https://internal.s3-cn-wlcb.ufileos.com/") {
		out = append(out, s.Phase)
	}
	return out
}

func TestHealthSelectChecks(t *testing.T) {
	assert.Equal(t, []benchmark.Phase{"internet " + defaultInternetURL, "intranet https://internal.s3-cn-wlcb.ufileos.com/"},
		healthPhases(t, "--only", "internet,intranet"))

	assert.Equal(t, []benchmark.Phase{"exec", "intranet https://internal.s3-cn-wlcb.ufileos.com/"},
		healthPhases(t, "--only", "intranet", "--only", "exec"),
		"checks run in their fixed order, whatever the order given")

	assert.Equal(t, []benchmark.Phase{"pause", "resume"}, healthPhases(t, "--only", "pause"),
		"pause covers resume")

	assert.Equal(t, []benchmark.Phase{
		"exec", "filesystem", "internet " + defaultInternetURL, "intranet https://internal.s3-cn-wlcb.ufileos.com/",
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

func TestHealthResolveIntranetURL(t *testing.T) {
	cfg := func(region, domain string) *config.Config {
		return &config.Config{Region: region, Domain: domain}
	}

	host, err := parseHealth(t).resolveIntranetURL(cfg("", "cn-sh.sandbox.ucloudai.com"))
	require.NoError(t, err)
	assert.Equal(t, "https://internal.s3-cn-sh2.ufileos.com/", host, "the region comes from the domain the API calls go to")

	host, err = parseHealth(t).resolveIntranetURL(cfg("us-ca", ""))
	require.NoError(t, err)
	assert.Equal(t, "https://internal.s3-us-ca.ufileos.com/", host)

	_, err = parseHealth(t).resolveIntranetURL(cfg("", "sandbox.example.com"))
	require.Error(t, err, "a custom domain alone does not tell the region")
	assert.Contains(t, err.Error(), "--intranet-url")

	host, err = parseHealth(t, "--intranet-url", "http://10.0.0.1:8080/").resolveIntranetURL(cfg("", "sandbox.example.com"))
	require.NoError(t, err)
	assert.Equal(t, "http://10.0.0.1:8080/", host, "the flag wins")

	host, err = parseHealth(t, "--skip", "intranet").resolveIntranetURL(cfg("", "sandbox.example.com"))
	require.NoError(t, err, "the region is not needed when the intranet check is skipped")
	assert.Empty(t, host)
}
