package volume

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/list"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/volume/v2/api"
)

const testID = "c8fd09b4-a581-4b69-b087-97b97a49ec4f"
const secondID = "b59f9f21-23d6-4e7b-96f6-7814722fc455"
const volumeJSON = `{"volumeID":"` + testID + `","name":"data","sizeGiB":500,"mountIP":"10.0.0.1","extraParams":{},"createdAt":"2026-11-20T01:02:03Z","updatedAt":"2026-11-20T01:02:03Z"}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func operationContext(t *testing.T, op cmd.Operation, args []string, handler func(*http.Request) (int, string)) cmd.OperationContext {
	t.Helper()
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))
	require.NoError(t, c.Args(c, c.Flags().Args()))
	require.NoError(t, c.ValidateRequiredFlags())
	cli, err := client.New(client.Options{
		APIKey: "test-key", APIURL: "https://api.example.com/gateway",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
			status, body := handler(r)
			return &http.Response{
				StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(body)), Request: r,
			}, nil
		})},
	})
	require.NoError(t, err)
	return cmd.OperationContext{Context: context.Background(), Cmd: c, Args: c.Flags().Args(), Client: cli}
}

func TestCreateRequest(t *testing.T) {
	for _, extra := range []bool{false, true} {
		t.Run(fmt.Sprint(extra), func(t *testing.T) {
			op := &createOperation{}
			args := []string{"data", "--size-gib", "500"}
			if extra {
				args = append(args, "--extra-params", `{"storageType":"Advanced","custom":true}`, "--idempotency-key", "create-data")
			}
			called := false
			ctx := operationContext(t, op, args, func(r *http.Request) (int, string) {
				called = true
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/gateway/v2/volumes", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, "data", body["name"])
				assert.Equal(t, float64(500), body["sizeGiB"])
				if extra {
					assert.Equal(t, map[string]any{"storageType": "Advanced", "custom": true}, body["extraParams"])
					assert.Equal(t, "create-data", r.Header.Get("Idempotency-Key"))
					return http.StatusOK, volumeJSON
				}
				assert.NotContains(t, body, "extraParams")
				assert.Empty(t, r.Header.Get("Idempotency-Key"))
				return http.StatusCreated, volumeJSON
			})
			require.NoError(t, op.Run(ctx))
			assert.True(t, called)
		})
	}
}

func TestInvalidArgumentsDoNotSendRequests(t *testing.T) {
	tests := []struct {
		name string
		op   cmd.Operation
		args []string
	}{
		{"empty name", &createOperation{}, []string{" ", "--size-gib", "500"}},
		{"zero create size", &createOperation{}, []string{"data", "--size-gib", "0"}},
		{"negative resize size", &resizeOperation{}, []string{testID, "--size-gib", "-1"}},
		{"invalid get UUID", &getOperation{}, []string{"not-a-uuid"}},
		{"invalid resize UUID", &resizeOperation{}, []string{"not-a-uuid", "--size-gib", "1000"}},
		{"invalid second delete UUID", &deleteOperation{}, []string{testID, "not-a-uuid"}},
		{"invalid list format", &listOperation{}, []string{"--format", "yaml"}},
		{"negative page", &listOperation{}, []string{"--page", "-1"}},
		{"negative limit", &listOperation{}, []string{"--limit", "-1"}},
	}
	for _, extra := range []string{"", "null", "[]", "true", "invalid", `{"key":`} {
		tests = append(tests, struct {
			name string
			op   cmd.Operation
			args []string
		}{"invalid extra " + extra, &createOperation{}, []string{"data", "--size-gib", "500", "--extra-params", extra}})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := operationContext(t, tt.op, tt.args, func(*http.Request) (int, string) {
				t.Error("invalid arguments must not send requests")
				return http.StatusInternalServerError, ""
			})
			require.Error(t, tt.op.Run(ctx))
		})
	}
}

func TestCreateAndResizeRequireSize(t *testing.T) {
	for _, op := range []cmd.Operation{&createOperation{}, &resizeOperation{}} {
		c := op.Command()
		require.NoError(t, c.ParseFlags([]string{testID}))
		assert.ErrorContains(t, c.ValidateRequiredFlags(), "size-gib")
	}
}

func TestGetAndResizeRequests(t *testing.T) {
	for _, resize := range []bool{false, true} {
		t.Run(fmt.Sprint(resize), func(t *testing.T) {
			var op cmd.Operation = &getOperation{}
			args := []string{testID}
			if resize {
				op = &resizeOperation{}
				args = append(args, "--size-gib", "1000")
			}
			called := false
			ctx := operationContext(t, op, args, func(r *http.Request) (int, string) {
				called = true
				assert.Equal(t, "/gateway/v2/volumes/"+testID, r.URL.Path)
				if resize {
					assert.Equal(t, "PATCH", r.Method)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, map[string]any{"sizeGiB": float64(1000)}, body)
				} else {
					assert.Equal(t, "GET", r.Method)
				}
				return http.StatusOK, volumeJSON
			})
			require.NoError(t, op.Run(ctx))
			assert.True(t, called)
		})
	}
}

func TestDeleteStopsAtFirstFailure(t *testing.T) {
	op := &deleteOperation{}
	var paths []string
	ctx := operationContext(t, op, []string{testID, secondID, testID}, func(r *http.Request) (int, string) {
		assert.Equal(t, "DELETE", r.Method)
		paths = append(paths, r.URL.Path)
		if len(paths) == 2 {
			return http.StatusConflict, `{"code":409,"message":"cleanup failed"}`
		}
		return http.StatusNoContent, ""
	})
	err := op.Run(ctx)
	require.ErrorIs(t, err, errdefs.ErrConflict)
	assert.Contains(t, err.Error(), secondID)
	assert.Equal(t, []string{"/gateway/v2/volumes/" + testID, "/gateway/v2/volumes/" + secondID}, paths)
}

func TestListPagination(t *testing.T) {
	for _, tt := range []struct {
		name       string
		args       []string
		total      int
		batchSize  int
		wantCount  int
		wantOffset []int
		hasMore    bool
	}{
		{"all", nil, 205, 100, 205, []int{0, 100, 200}, false},
		{"second page", []string{"-p", "2", "-l", "10"}, 205, 100, 10, []int{10}, true},
		{"large page", []string{"-l", "150"}, 205, 100, 150, []int{0, 100}, true},
		{"large second page", []string{"-p", "2", "-l", "150"}, 310, 100, 150, []int{150, 250}, true},
		{"last page", []string{"-p", "3", "-l", "100"}, 205, 100, 5, []int{200}, false},
		{"past end", []string{"-p", "4", "-l", "100"}, 205, 100, 0, []int{300}, false},
		{"empty", nil, 0, 100, 0, []int{0}, false},
		{"partial batches", nil, 125, 60, 125, []int{0, 60, 120}, false},
		{"empty batch", nil, 125, 0, 0, []int{0}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			op := &listOperation{}
			var offsets []int
			ctx := operationContext(t, op, tt.args, func(r *http.Request) (int, string) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/gateway/v2/volumes", r.URL.Path)
				offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
				require.NoError(t, err)
				limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
				require.NoError(t, err)
				assert.Greater(t, limit, 0)
				assert.LessOrEqual(t, limit, 100)
				offsets = append(offsets, offset)
				items := make([]api.Volume, max(0, min(limit, tt.batchSize, tt.total-offset)))
				for i := range items {
					items[i] = api.Volume{VolumeID: uuid.MustParse(testID), Name: fmt.Sprintf("data-%d", offset+i)}
				}
				body, err := json.Marshal(api.VolumeList{Items: items, Total: int64(tt.total)})
				require.NoError(t, err)
				return http.StatusOK, string(body)
			})
			page, err := op.fetchPage(ctx)
			require.NoError(t, err)
			assert.Len(t, page.Items, tt.wantCount)
			assert.Equal(t, int64(tt.total), page.Total)
			assert.Equal(t, tt.hasMore, page.HasMore)
			assert.Equal(t, tt.wantOffset, offsets)
			if len(page.Items) > 0 {
				assert.Equal(t, fmt.Sprintf("data-%d", tt.wantOffset[0]), page.Items[0].Name)
			}
		})
	}
}

func TestListDefaults(t *testing.T) {
	op := &listOperation{}
	require.NoError(t, op.Command().ParseFlags(nil))
	assert.Equal(t, list.Options{Page: 1, Limit: 0, Format: list.FormatPretty}, op.list)
}

func TestCommandArguments(t *testing.T) {
	for _, op := range []cmd.Operation{&createOperation{}, &getOperation{}, &resizeOperation{}, &deleteOperation{}} {
		c := op.Command()
		assert.Error(t, c.Args(c, nil), c.Name())
		assert.NoError(t, c.Args(c, []string{testID}), c.Name())
		if c.Name() != "delete" {
			assert.Error(t, c.Args(c, []string{testID, secondID}), c.Name())
		}
	}
	c := (&listOperation{}).Command()
	assert.Error(t, c.Args(c, []string{"extra"}))
}
