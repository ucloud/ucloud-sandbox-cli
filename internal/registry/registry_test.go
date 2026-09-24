package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	tests := []struct {
		name  string
		image string
		want  string
	}{
		{name: "a bare name is Docker Hub", image: "alpine", want: DefaultDomain},
		{name: "a namespaced name is Docker Hub", image: "library/alpine:3.20", want: DefaultDomain},
		{name: "an explicit registry", image: "uhub.service.ucloud.cn/team/base:latest", want: "uhub.service.ucloud.cn"},
		{name: "a registry with a port", image: "localhost:5000/app", want: "localhost:5000"},
		{name: "a digest reference", image: "ghcr.io/owner/app@sha256:" + zeroDigest, want: "ghcr.io"},
		{name: "surrounding space is ignored", image: "  ghcr.io/owner/app  ", want: "ghcr.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Domain(tt.image)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// zeroDigest is a syntactically valid sha256 digest.
const zeroDigest = "0000000000000000000000000000000000000000000000000000000000000000"

func TestDomainRejectsAnInvalidReference(t *testing.T) {
	// A repository path has to be lower-case; a first component that is not is
	// read as a registry host instead, which is why "UPPERCASE/app" is fine.
	for _, image := range []string{"", "   ", "a::b", "Alpine", "ghcr.io/Owner/app"} {
		_, err := Domain(image)
		assert.Error(t, err, "image %q", image)
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{name: "plain", domain: "docker.io", want: "docker.io"},
		{name: "upper case", domain: "GHCR.IO", want: "ghcr.io"},
		{name: "https scheme", domain: "https://ghcr.io", want: "ghcr.io"},
		{name: "http scheme", domain: "http://localhost:5000", want: "localhost:5000"},
		{name: "trailing slash", domain: "https://ghcr.io/", want: "ghcr.io"},
		{name: "surrounding space", domain: "  ghcr.io ", want: "ghcr.io"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeDomain(tt.domain)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeDomainRejectsSomethingThatIsNotADomain(t *testing.T) {
	// An image reference where a registry was expected would be stored under a
	// key no lookup ever matches.
	for _, domain := range []string{"", "   ", "ghcr.io/owner/app", "ghcr.io app"} {
		_, err := NormalizeDomain(domain)
		assert.Error(t, err, "domain %q", domain)
	}
}

// A domain must not be run through Domain: reference parsing reads it as a
// Docker Hub repository and answers docker.io, which is why the two have
// separate entry points.
func TestDomainIsNotUsableForABareDomain(t *testing.T) {
	got, err := Domain("myregistry.example.com")
	require.NoError(t, err)
	assert.Equal(t, DefaultDomain, got, "this is exactly the trap NormalizeDomain exists to avoid")

	normalized, err := NormalizeDomain("myregistry.example.com")
	require.NoError(t, err)
	assert.Equal(t, "myregistry.example.com", normalized)
}
