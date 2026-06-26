package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_TimeOnly(t *testing.T) {
	now := time.Now()

	cases := []struct{ input, layout string }{
		{"14:30", "15:04"},
		{"14:30:59", "15:04:05"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			ref, _ := time.ParseInLocation(tc.layout, tc.input, time.Local)
			got, err := Parse(tc.input)
			require.NoError(t, err)
			assert.Equal(t, now.Year(), got.Year())
			assert.Equal(t, now.Month(), got.Month())
			assert.Equal(t, now.Day(), got.Day())
			assert.Equal(t, ref.Hour(), got.Hour())
			assert.Equal(t, ref.Minute(), got.Minute())
			assert.Equal(t, ref.Second(), got.Second())
		})
	}
}

func TestParse_MonthDay(t *testing.T) {
	now := time.Now()

	cases := []struct {
		input  string
		month  time.Month
		day    int
		hour   int
		minute int
		second int
	}{
		{"06-23 12:00", time.June, 23, 12, 0, 0},
		{"06-23 12:00:45", time.June, 23, 12, 0, 45},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Parse(tc.input)
			require.NoError(t, err)
			assert.Equal(t, now.Year(), got.Year())
			assert.Equal(t, tc.month, got.Month())
			assert.Equal(t, tc.day, got.Day())
			assert.Equal(t, tc.hour, got.Hour())
			assert.Equal(t, tc.minute, got.Minute())
			assert.Equal(t, tc.second, got.Second())
		})
	}
}

func TestParse_FullDate(t *testing.T) {
	cases := []struct {
		input string
		year  int
		month time.Month
		day   int
		hour  int
	}{
		{"2025-07-23 12:00", 2025, time.July, 23, 12},
		{"2025-07-23 12:00:01", 2025, time.July, 23, 12},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := Parse(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.year, got.Year())
			assert.Equal(t, tc.month, got.Month())
			assert.Equal(t, tc.day, got.Day())
			assert.Equal(t, tc.hour, got.Hour())
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse("not-a-date")
	assert.Error(t, err)
}

func TestParseSince(t *testing.T) {
	before := time.Now()
	got, err := ParseSince("1h")
	after := time.Now()
	require.NoError(t, err)
	assert.True(t, got.After(before.Add(-time.Hour-time.Second)))
	assert.True(t, got.Before(after.Add(-time.Hour+time.Second)))
}

func TestParseSince_Invalid(t *testing.T) {
	_, err := ParseSince("bad")
	assert.Error(t, err)
}
