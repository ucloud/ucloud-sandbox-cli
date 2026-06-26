package template

import (
	"context"
	"fmt"

	"github.com/manifoldco/promptui"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// resolveTargets determines which templates to operate on based on args, flags, or interactive selection.
// Returns: template IDs, optional local config (if loaded), error
func resolveTargets(ctx context.Context, client *sdk.Client, args []string, path string, selectMode bool) ([]string, *LocalConfig, error) {
	// 1. From args
	if len(args) > 0 {
		return args, nil, nil
	}

	// 2. From interactive selection
	if selectMode {
		paginator := client.ListTemplates(ctx)
		var templates []sdk.TemplateInfo
		for paginator.HasNext() {
			items, err := paginator.NextItems(ctx)
			if err != nil {
				return nil, nil, err
			}
			templates = append(templates, items...)
		}

		if len(templates) == 0 {
			return nil, nil, fmt.Errorf("no templates available")
		}

		// Use promptui for selection
		sel := promptui.Select{
			Label: "Select template",
			Items: templates,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   "▸ {{ .TemplateID }}",
				Inactive: "  {{ .TemplateID }}",
				Selected: "✔ {{ .TemplateID }}",
			},
		}

		idx, _, err := sel.Run()
		if err != nil {
			return nil, nil, err
		}

		return []string{templates[idx].TemplateID}, nil, nil
	}

	// 3. From local config
	cfg, err := loadConfig(path)
	if err != nil || cfg.TemplateID == "" {
		return nil, nil, fmt.Errorf("no template specified; use [template] argument, -s flag, or local config")
	}

	return []string{cfg.TemplateID}, cfg, nil
}
