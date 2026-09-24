package sandbox

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// parseKillFlags registers the kill flags the way the command does and returns
// the operation they filled in, along with the remaining arguments.
func parseKillFlags(t *testing.T, args ...string) (*killOperation, []string) {
	t.Helper()

	op := &killOperation{}
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))

	return op, c.Flags().Args()
}

func TestKillValidate(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		flags   []string
		wantErr string
	}{
		{
			name:    "neither IDs nor --all",
			wantErr: "specify sandbox IDs or use --all",
		},
		{
			name:    "IDs together with --all",
			args:    []string{"sbx-1"},
			flags:   []string{"--all"},
			wantErr: "cannot use --all together with sandbox IDs",
		},
		{
			name:    "negative --limit",
			flags:   []string{"--all", "--limit", "-1"},
			wantErr: "must be 0 or greater",
		},
		{
			name: "explicit IDs",
			args: []string{"sbx-1", "sbx-2"},
		},
		{
			name:  "--all on its own",
			flags: []string{"--all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op, _ := parseKillFlags(t, tt.flags...)

			err := op.validate(tt.args)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestKillFlags(t *testing.T) {
	op, args := parseKillFlags(t,
		"-a", "-y",
		"--limit", "5",
		"-s", "running",
		"-t", "system/base",
		"-m", "env=dev",
	)

	assert.True(t, op.all)
	assert.True(t, op.yes)
	assert.Equal(t, 5, op.limit)

	// The filters come from the flags list shares with this command.
	require.NotNil(t, op.params.State)
	assert.Equal(t, []api.SandboxState{"running"}, *op.params.State)
	require.NotNil(t, op.params.Template)
	assert.Equal(t, "system/base", *op.params.Template)
	require.NotNil(t, op.params.Metadata)
	assert.Equal(t, "env=dev", *op.params.Metadata)

	assert.Empty(t, args)
}

func TestKillFiltersStayUnsetByDefault(t *testing.T) {
	op, _ := parseKillFlags(t, "--all")

	assert.Nil(t, op.params.State)
	assert.Nil(t, op.params.Template)
	assert.Nil(t, op.params.Metadata)
	assert.Nil(t, op.params.Order)
	assert.Nil(t, op.params.StartedAfter)
	assert.Equal(t, 0, op.limit, "no limit kills every match")
}

// pagesOf returns a paginator handing out the given pages in order.
func pagesOf(pages ...[]string) *transport.Paginator[api.ListedSandbox] {
	index := 0

	return transport.NewPaginator(func(_ context.Context, _ string) ([]api.ListedSandbox, string, error) {
		if index >= len(pages) {
			return nil, "", nil
		}

		items := make([]api.ListedSandbox, 0, len(pages[index]))
		for _, id := range pages[index] {
			items = append(items, api.ListedSandbox{SandboxID: id})
		}

		index++

		token := ""
		if index < len(pages) {
			token = fmt.Sprintf("page-%d", index)
		}

		return items, token, nil
	})
}

func TestKillMatchingIDs(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		pages [][]string
		want  []string
	}{
		{
			name:  "no limit walks every page",
			pages: [][]string{{"sbx-1", "sbx-2"}, {"sbx-3"}},
			want:  []string{"sbx-1", "sbx-2", "sbx-3"},
		},
		{
			name:  "the limit cuts within a page",
			limit: 2,
			pages: [][]string{{"sbx-1", "sbx-2", "sbx-3"}},
			want:  []string{"sbx-1", "sbx-2"},
		},
		{
			name:  "the limit stops fetching further pages",
			limit: 2,
			pages: [][]string{{"sbx-1", "sbx-2"}, {"sbx-3"}},
			want:  []string{"sbx-1", "sbx-2"},
		},
		{
			name:  "a limit above the match count keeps everything",
			limit: 10,
			pages: [][]string{{"sbx-1"}},
			want:  []string{"sbx-1"},
		},
		{
			name:  "nothing matched",
			pages: [][]string{{}},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := &killOperation{all: true, limit: tt.limit}

			ids, err := op.matchingIDs(t.Context(), pagesOf(tt.pages...))
			require.NoError(t, err)
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestKillMatchingIDsPropagatesError(t *testing.T) {
	wanted := fmt.Errorf("listing failed")

	paginator := transport.NewPaginator(func(_ context.Context, _ string) ([]api.ListedSandbox, string, error) {
		return nil, "", wanted
	})

	op := &killOperation{all: true}

	_, err := op.matchingIDs(t.Context(), paginator)
	assert.ErrorIs(t, err, wanted)
}
