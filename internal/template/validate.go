package template

import (
	"fmt"
	"regexp"
)

var nameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$`)

// ValidateName validates template name format.
// Template names must contain only lowercase letters, numbers, hyphens, and underscores,
// and cannot start or end with a hyphen or underscore.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}
	if !nameRegex.MatchString(name) {
		return fmt.Errorf("template name must contain only lowercase letters, numbers, hyphens, and underscores, and cannot start or end with a hyphen or underscore")
	}
	return nil
}
