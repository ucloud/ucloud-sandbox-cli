package prompt

import (
	"errors"

	"github.com/manifoldco/promptui"
	"github.com/ucloud/ucloud-sandbox-cli/internal/template"
)

var predefinedRegions = []string{"cn-wlcb", "us-ca"}

// AskAPIKey prompts for an API key with masked input.
func AskAPIKey() (string, error) {
	p := promptui.Prompt{
		Label: "API Key",
		Mask:  '*',
		Validate: func(s string) error {
			if s == "" {
				return errors.New("API key cannot be empty")
			}
			return nil
		},
	}
	return p.Run()
}

// AskRegion prompts the user to select or enter a region.
// If allowSkip is true, an additional "Skip" option is shown.
func AskRegion(allowSkip bool) (string, error) {
	items := append([]string{}, predefinedRegions...)
	items = append(items, "Custom")
	if allowSkip {
		items = append(items, "Skip")
	}

	sel := promptui.Select{
		Label: "Region",
		Items: items,
	}
	_, choice, err := sel.Run()
	if err != nil {
		return "", err
	}

	switch choice {
	case "Skip":
		return "", nil
	case "Custom":
		return askCustomRegion()
	default:
		return choice, nil
	}
}

func askCustomRegion() (string, error) {
	p := promptui.Prompt{
		Label: "Custom region",
		Validate: func(s string) error {
			if s == "" {
				return errors.New("region cannot be empty")
			}
			return nil
		},
	}
	return p.Run()
}

// AskTemplateName prompts for a template name with validation.
func AskTemplateName(defaultName string) (string, error) {
	p := promptui.Prompt{
		Label:   "Template name",
		Default: defaultName,
		Validate: func(s string) error {
			return template.ValidateName(s)
		},
	}
	return p.Run()
}

// Confirm shows a yes/no confirmation prompt.
func Confirm(label string) (bool, error) {
	p := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}
	_, err := p.Run()
	if err != nil {
		// User pressed 'n' or Ctrl+C
		if err == promptui.ErrAbort {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
