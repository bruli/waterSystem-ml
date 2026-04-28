package ml

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewExecution(t *testing.T) {
	now := time.Now()
	type args struct {
		zone          string
		executionTime time.Time
	}
	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "with a recent execution time, then it returns true",
			args: args{
				zone:          "zone",
				executionTime: now.Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "with a old execution time, then it returns false",
			args: args{
				zone:          "zone",
				executionTime: now.Add(-3 * time.Hour),
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Execution struct,`+tt.name, func(t *testing.T) {
			t.Parallel()
			exec := NewExecution(tt.args.zone, tt.args.executionTime)
			require.Equal(t, tt.expected, exec.IsRecentlyExecuted())
		})
	}
}
