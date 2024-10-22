package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	hour := int64(time.Hour)
	day := 24 * hour
	testCases := []struct {
		str string
		val int64
	}{
		{"0", 0},
		{"-", 0},
		{"0s", 0},
		{"1m", int64(time.Minute)},
		{"1h", hour},
		{"24h", 24 * hour},
		{"1d", day},
		{"1w", 7 * day},
		{"4w", 4 * 7 * day},
		{"1M", 30 * day},
		{"1y", 365 * day},
		{"14M", 14 * 30 * day},
		{"1y2M", 1*365*day + 2*30*day},
		{"+1y2M", 1*365*day + 2*30*day},
		{"-1y2M", -1 * (1*365*day + 2*30*day)},
		{"1y2M3w4d", 1*365*day + 2*30*day + 3*7*day + 4*day},
		{"1y8h16m", 1*365*day + 8*hour + int64(16*time.Minute)},
		{"4y8h16m30s", 4*365*day + 8*hour + int64(16*time.Minute) + int64(30*time.Second)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.str, func(t *testing.T) {
			assert := require.New(t)
			d, err := ParseDuration(testCase.str)
			assert.NoError(err)
			assert.Equal(Duration(testCase.val), d)
		})
	}
}
