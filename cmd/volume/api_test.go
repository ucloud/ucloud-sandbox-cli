package volume

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

const testID = "c8fd09b4-a581-4b69-b087-97b97a49ec4f"
const secondID = "b59f9f21-23d6-4e7b-96f6-7814722fc455"
const volumeJSON = `{"volumeID":"` + testID + `","name":"data","totalSizeGiB":10,"usedSizeGiB":0.25,"availableSizeGiB":9.75}`
const legacyJSON = `{"volumeID":"` + secondID + `","name":"legacy","token":"legacy-token"}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func operationContext(t *testing.T, op cmd.Operation, args []string, handler func(*http.Request) (int, string)) (cmd.OperationContext, *bytes.Buffer) {
	t.Helper()
	c := op.Command()
	require.NoError(t, c.ParseFlags(args))
	require.NoError(t, c.Args(c, c.Flags().Args()))
	require.NoError(t, c.ValidateRequiredFlags())
	out := new(bytes.Buffer)
	c.SetOut(out)
	cli, err := client.New(client.Options{
		APIKey: "test-key", APIURL: "https://api.example.com/gateway",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, "api.example.com", r.URL.Host)
			assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
			status, body := handler(r)
			return &http.Response{
				StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(body)), Request: r,
			}, nil
		})},
	})
	require.NoError(t, err)
	return cmd.OperationContext{Context: context.Background(), Cmd: c, Args: c.Flags().Args(), Client: cli}, out
}

func TestCreateRequest(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		op := &createOperation{}
		args := []string{"data"}
		want := `{"name":"data"}`
		if explicit {
			args = append(args, "--size-gib", "10")
			want = `{"name":"data","totalSizeGiB":10}`
		}
		called := false
		ctx, out := operationContext(t, op, args, func(r *http.Request) (int, string) {
			called = true
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/gateway/volumes", r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.JSONEq(t, want, string(body))
			return http.StatusCreated, volumeJSON
		})
		require.NoError(t, op.Run(ctx))
		assert.True(t, called)
		assert.Contains(t, out.String(), testID)
	}
}

func TestGetPreservesNewAndLegacyFields(t *testing.T) {
	for _, body := range []string{volumeJSON, legacyJSON} {
		op := &getOperation{}
		ctx, out := operationContext(t, op, []string{testID}, func(r *http.Request) (int, string) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/gateway/volumes/"+testID, r.URL.Path)
			return http.StatusOK, body
		})
		require.NoError(t, op.Run(ctx))
		assert.JSONEq(t, body, out.String())
	}
}

func TestResizeRequest(t *testing.T) {
	op := &resizeOperation{}
	ctx, out := operationContext(t, op, []string{testID, "--size-gib", "20"}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/gateway/volumes/"+testID+"/resize", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"totalSizeGiB":20}`, string(body))
		return http.StatusOK, strings.Replace(volumeJSON, `"totalSizeGiB":10`, `"totalSizeGiB":20`, 1)
	})
	require.NoError(t, op.Run(ctx))
	assert.Contains(t, out.String(), `"totalSizeGiB": 20`)
}

func TestMixedVolumeList(t *testing.T) {
	for _, format := range []string{"pretty", "json"} {
		op := &listOperation{}
		ctx, out := operationContext(t, op, []string{"--format", format}, func(r *http.Request) (int, string) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/gateway/volumes", r.URL.Path)
			assert.Empty(t, r.URL.RawQuery)
			return http.StatusOK, "[" + volumeJSON + "," + legacyJSON + "]"
		})
		require.NoError(t, op.Run(ctx))
		if format == "json" {
			assert.JSONEq(t, "["+volumeJSON+","+legacyJSON+"]", out.String())
			continue
		}
		for _, value := range []string{"Total (GiB)", "Used (GiB)", "Available (GiB)", "10", "0.25", "9.75", "legacy", "-"} {
			assert.Contains(t, out.String(), value)
		}
		assert.NotContains(t, out.String(), "legacy-token")
	}
}

func TestListPaginationKeepsCapacityFields(t *testing.T) {
	op := &listOperation{}
	ctx, out := operationContext(t, op, []string{"--page", "2", "--limit", "1", "--format", "json"}, func(*http.Request) (int, string) {
		return http.StatusOK, "[" + legacyJSON + "," + volumeJSON + "]"
	})
	require.NoError(t, op.Run(ctx))
	assert.JSONEq(t, "["+volumeJSON+"]", out.String())
}

func TestInvalidArgumentsDoNotSendRequests(t *testing.T) {
	for _, tt := range []struct {
		op   cmd.Operation
		args []string
	}{
		{&createOperation{}, []string{" "}},
		{&createOperation{}, []string{"data", "--size-gib", "0"}},
		{&createOperation{}, []string{"data", "--size-gib", "-1"}},
		{&resizeOperation{}, []string{testID, "--size-gib", "0"}},
		{&resizeOperation{}, []string{testID, "--size-gib", "-1"}},
		{&getOperation{}, []string{"not-a-uuid"}},
		{&resizeOperation{}, []string{"not-a-uuid", "--size-gib", "20"}},
		{&deleteOperation{}, []string{testID, "not-a-uuid"}},
	} {
		ctx, _ := operationContext(t, tt.op, tt.args, func(*http.Request) (int, string) {
			t.Fatal("invalid input must not send requests")
			return 0, ""
		})
		require.Error(t, tt.op.Run(ctx))
	}
}

func TestResizeRequiresSize(t *testing.T) {
	c := (&resizeOperation{}).Command()
	require.NoError(t, c.ParseFlags([]string{testID}))
	assert.ErrorContains(t, c.ValidateRequiredFlags(), "size-gib")
}

func TestDeleteRequests(t *testing.T) {
	op := &deleteOperation{}
	calls := 0
	ctx, out := operationContext(t, op, []string{testID, secondID}, func(r *http.Request) (int, string) {
		assert.Equal(t, http.MethodDelete, r.Method)
		calls++
		if calls == 1 {
			assert.Equal(t, "/gateway/volumes/"+testID, r.URL.Path)
			return http.StatusNoContent, ""
		}
		assert.Equal(t, "/gateway/volumes/"+secondID, r.URL.Path)
		return http.StatusNotFound, `{"code":404,"message":"missing"}`
	})
	require.NoError(t, op.Run(ctx))
	assert.Equal(t, 2, calls)
	assert.Contains(t, out.String(), "deleted")
	assert.Contains(t, out.String(), "not found")
}

func TestResizePropagatesLegacyOrShrinkRejection(t *testing.T) {
	op := &resizeOperation{}
	ctx, out := operationContext(t, op, []string{testID, "--size-gib", "1"}, func(*http.Request) (int, string) {
		return http.StatusBadRequest, `{"code":400,"message":"volume cannot be resized"}`
	})
	require.ErrorIs(t, op.Run(ctx), errdefs.ErrInvalidArgument)
	assert.Empty(t, out.String())
}
