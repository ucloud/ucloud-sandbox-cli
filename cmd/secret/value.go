package secret

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"golang.org/x/term"
)

// valueSource decides where a secret's value comes from. The value is not a
// positional argument on purpose: an argument ends up in the shell's history
// and in the process list of everyone on the machine.
type valueSource struct {
	value string
	stdin bool

	// fs tells an explicitly empty --value apart from an absent one.
	fs *pflag.FlagSet
}

// AddFlags registers the flags naming where the value comes from.
func (s *valueSource) AddFlags(fs *pflag.FlagSet) {
	s.fs = fs

	fs.StringVarP(&s.value, "value", "", "", "The secret's value")
	fs.BoolVarP(&s.stdin, "value-stdin", "", false, "Read the secret's value from stdin")
}

// given reports whether --value was provided, an empty value included.
func (s *valueSource) given() bool {
	return s.fs != nil && s.fs.Changed("value")
}

// Read returns the value, asking for it under label when no flag named it.
func (s *valueSource) Read(label string) (string, error) {
	if s.stdin {
		if s.given() {
			return "", fmt.Errorf("--value and --value-stdin are mutually exclusive")
		}

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read the value from stdin: %w", err)
		}

		// A value piped in from a file or a heredoc carries the newline that
		// ended it, which is not part of the secret.
		return strings.TrimRight(string(data), "\r\n"), nil
	}

	if s.given() {
		return s.value, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("no value given: use --value or --value-stdin")
	}

	return prompt.AskSecret(label)
}
