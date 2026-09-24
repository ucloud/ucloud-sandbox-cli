// Package registry resolves which container registry an image comes from, so
// the right credentials can be picked for it.
package registry

import (
	"fmt"
	"strings"

	"github.com/distribution/reference"
)

// DefaultDomain is the registry a bare image name refers to.
const DefaultDomain = "docker.io"

// Domain returns the registry an image reference is pulled from:
//
//	alpine                                  -> docker.io
//	uhub.service.ucloud.cn/team/base:latest -> uhub.service.ucloud.cn
//	localhost:5000/app                      -> localhost:5000
//
// Pass a full image reference. A bare domain is not one: reference parsing
// reads "myregistry.example.com" as a Docker Hub repository and would answer
// docker.io. Use NormalizeDomain for a domain the user typed.
func Domain(image string) (string, error) {
	named, err := reference.ParseNormalizedNamed(strings.TrimSpace(image))
	if err != nil {
		return "", fmt.Errorf("invalid image reference %q: %w", image, err)
	}

	return reference.Domain(named), nil
}

// NormalizeDomain cleans up a registry domain the user typed, dropping a
// scheme and any trailing slashes and lower-casing the result, so the same
// registry is stored under one key however it was written.
func NormalizeDomain(domain string) (string, error) {
	cleaned := strings.TrimSpace(domain)

	// A domain pasted out of a browser or a docker login line may carry one.
	for _, scheme := range []string{"https://", "http://"} {
		cleaned = strings.TrimPrefix(cleaned, scheme)
	}

	cleaned = strings.Trim(cleaned, "/")
	cleaned = strings.ToLower(cleaned)

	if cleaned == "" {
		return "", fmt.Errorf("registry domain cannot be empty")
	}

	// A path means an image reference was given where a registry was expected,
	// which would otherwise be stored as a key nothing ever matches.
	if strings.ContainsAny(cleaned, "/ \t") {
		return "", fmt.Errorf("invalid registry domain %q: expected a host such as %s", domain, DefaultDomain)
	}

	return cleaned, nil
}
