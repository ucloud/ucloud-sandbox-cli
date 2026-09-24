package flags

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

func TestStringMapVarPAllocatesOnFirstUse(t *testing.T) {
	var envVars api.EnvVars

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	StringMapVarP(fs, &envVars, "env", "e", "")

	require.Nil(t, envVars, "target stays nil until the flag is provided")

	require.NoError(t, fs.Parse([]string{"-e", "FOO: bar", "--env", "BAZ:qux"}))

	assert.Equal(t, api.EnvVars{"FOO": "bar", "BAZ": "qux"}, envVars)
}

func TestNullableStringMapVarPAllocatesOnFirstUse(t *testing.T) {
	var envVars *api.EnvVars

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	NullableStringMapVarP(fs, &envVars, "env", "e", "")

	require.Nil(t, envVars, "target stays nil until the flag is provided")

	require.NoError(t, fs.Parse([]string{"-e", "FOO: bar", "--env", "BAZ:qux"}))

	require.NotNil(t, envVars)
	assert.Equal(t, api.EnvVars{"FOO": "bar", "BAZ": "qux"}, *envVars)
}

func TestStringMapVarPRejectsInvalidValue(t *testing.T) {
	var envVars api.EnvVars
	var nullableEnvVars *api.EnvVars

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	StringMapVarP(fs, &envVars, "env", "e", "")
	NullableStringMapVarP(fs, &nullableEnvVars, "nullable-env", "", "")

	assert.Error(t, fs.Parse([]string{"--env", "novalue"}))
	assert.Error(t, fs.Parse([]string{"--env", ": bar"}))
	assert.Error(t, fs.Parse([]string{"--nullable-env", "novalue"}))
	assert.Error(t, fs.Parse([]string{"--nullable-env", ": bar"}))
}

func TestNullableBoolVarP(t *testing.T) {
	var unset, explicit, noOptDef *bool

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	NullableBoolVarP(fs, &unset, "unset", "", "")
	NullableBoolVarP(fs, &explicit, "explicit", "", "")
	NullableBoolVarP(fs, &noOptDef, "no-opt-def", "", "")

	require.NoError(t, fs.Parse([]string{"--explicit=false", "--no-opt-def"}))

	assert.Nil(t, unset)
	require.NotNil(t, explicit)
	assert.False(t, *explicit)
	require.NotNil(t, noOptDef)
	assert.True(t, *noOptDef)
}

func TestNullableVarPWithNamedTypes(t *testing.T) {
	type state string
	type count int32
	type port uint32
	type tags []string
	type ports []uint32

	var s *state
	var c *count
	var p *port
	var tg *tags
	var ps *ports

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	NullableStringVarP(fs, &s, "state", "", "")
	NullableInt32VarP(fs, &c, "count", "", "")
	NullableUint32VarP(fs, &p, "port", "", "")
	NullableStringSliceVarP(fs, &tg, "tag", "", "")
	NullableUint32SliceVarP(fs, &ps, "https-port", "", "")

	require.NoError(t, fs.Parse([]string{
		"--state", "running",
		"--count", "7",
		"--port", "8080",
		"--tag", "a",
		"--https-port", "443",
	}))

	require.NotNil(t, s)
	assert.Equal(t, state("running"), *s)
	require.NotNil(t, c)
	assert.Equal(t, count(7), *c)
	require.NotNil(t, p)
	assert.Equal(t, port(8080), *p)
	require.NotNil(t, tg)
	assert.Equal(t, tags{"a"}, *tg)
	require.NotNil(t, ps)
	assert.Equal(t, ports{443}, *ps)
}

func TestNullableStringSliceVarPWithNamedElementType(t *testing.T) {
	var states *[]api.SandboxState

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	NullableStringSliceVarP(fs, &states, "state", "s", "")

	require.Nil(t, states, "target stays nil until the flag is provided")

	require.NoError(t, fs.Parse([]string{"-s", "running", "--state", "paused"}))

	require.NotNil(t, states)
	assert.Equal(t, []api.SandboxState{api.Running, api.Paused}, *states)
}

func TestNullableDatetimeVarP(t *testing.T) {
	var unset, value *time.Time

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	NullableDatetimeVarP(fs, &unset, "unset", "", "")
	NullableDatetimeVarP(fs, &value, "value", "", "")

	require.NoError(t, fs.Parse([]string{"--value", "2025-07-23 15:04"}))

	assert.Nil(t, unset)
	require.NotNil(t, value)
	assert.Equal(t, time.Date(2025, 7, 23, 15, 4, 0, 0, time.Local), *value)

	assert.Error(t, fs.Parse([]string{"--value", "yesterday"}))
}

func TestNullableUint32VarP(t *testing.T) {
	var unset, value *uint32

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	NullableUint32VarP(fs, &unset, "unset", "", "")
	NullableUint32VarP(fs, &value, "value", "", "")

	require.NoError(t, fs.Parse([]string{"--value", "0"}))

	assert.Nil(t, unset)
	require.NotNil(t, value)
	assert.Equal(t, uint32(0), *value)

	assert.Error(t, fs.Parse([]string{"--value", "-1"}))
	assert.Error(t, fs.Parse([]string{"--value", "4294967296"}))
	assert.Error(t, fs.Parse([]string{"--value", "abc"}))
}

func TestNullableInt64VarP(t *testing.T) {
	var unset, value *int64

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	NullableInt64VarP(fs, &unset, "unset", "", "")
	NullableInt64VarP(fs, &value, "value", "", "")

	require.Nil(t, value, "target stays nil until the flag is provided")

	require.NoError(t, fs.Parse([]string{"--value", "1758693600"}))

	assert.Nil(t, unset)
	require.NotNil(t, value)
	assert.Equal(t, int64(1758693600), *value)

	require.NoError(t, fs.Parse([]string{"--value", "-1"}))
	assert.Equal(t, int64(-1), *value)

	assert.Error(t, fs.Parse([]string{"--value", "9223372036854775808"}))
	assert.Error(t, fs.Parse([]string{"--value", "abc"}))
}

func TestNullableStringSliceVarPAppendsPerOccurrence(t *testing.T) {
	var unset, values *[]string

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	NullableStringSliceVarP(fs, &unset, "unset", "", "")
	NullableStringSliceVarP(fs, &values, "value", "v", "")

	require.Nil(t, values, "target stays nil until the flag is provided")

	// A comma is kept verbatim: only repeated flags add entries.
	require.NoError(t, fs.Parse([]string{"-v", "a,b", "--value", "c", "--value", ""}))

	assert.Nil(t, unset)
	require.NotNil(t, values)
	assert.Equal(t, []string{"a,b", "c", ""}, *values)
}

func TestNullableUint32SliceVarPAppendsPerOccurrence(t *testing.T) {
	var unset, values *[]uint32

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.SetOutput(nil)
	NullableUint32SliceVarP(fs, &unset, "unset", "", "")
	NullableUint32SliceVarP(fs, &values, "value", "v", "")

	require.Nil(t, values, "target stays nil until the flag is provided")

	require.NoError(t, fs.Parse([]string{"-v", "80", "--value", "443"}))

	assert.Nil(t, unset)
	require.NotNil(t, values)
	assert.Equal(t, []uint32{80, 443}, *values)

	assert.Error(t, fs.Parse([]string{"--value", "-1"}))
	assert.Error(t, fs.Parse([]string{"--value", "4294967296"}))
	assert.Error(t, fs.Parse([]string{"--value", "abc"}))
}
