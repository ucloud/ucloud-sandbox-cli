package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		// Valid names
		{"simple lowercase", "mytemplate", false},
		{"with numbers", "template123", false},
		{"with hyphens", "my-template", false},
		{"with underscores", "my_template", false},
		{"mixed", "my-template_123", false},
		{"single char", "a", false},
		{"single digit", "1", false},

		// Invalid names
		{"empty", "", true},
		{"starts with hyphen", "-mytemplate", true},
		{"ends with hyphen", "mytemplate-", true},
		{"starts with underscore", "_mytemplate", true},
		{"ends with underscore", "mytemplate_", true},
		{"uppercase", "MyTemplate", true},
		{"spaces", "my template", true},
		{"special chars", "my@template", true},
		{"dots", "my.template", true},
		{"only hyphen", "-", true},
		{"only underscore", "_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if tt.wantError {
				assert.Error(t, err, "expected error for input: %s", tt.input)
			} else {
				assert.NoError(t, err, "expected no error for input: %s", tt.input)
			}
		})
	}
}
