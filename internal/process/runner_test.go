package process

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
	"path/filepath"
	"testing"
	"time"
)

func TestRunTA(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	r := NewRunner(logsSink, l)
	defer r.Shutdown()
	scheduling, err := r.Run(filepath.Join("testdata", "ta"))
	require.NoError(t, err)
	require.True(t, scheduling)

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 20
	}, 2*time.Second, 10*time.Millisecond)
}

func TestRunPeriodic(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	r := NewRunner(logsSink, l)
	defer r.Shutdown()
	scheduling, err := r.Run(filepath.Join("testdata", "periodic"))
	require.NoError(t, err)
	require.True(t, scheduling)

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 10
	}, 1*time.Second, 10*time.Millisecond)

	result := make([]string, 0, 10)
	for _, log := range logsSink.AllLogs() {
		for _, rl := range log.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					result = append(result, lr.Body().Str())
				}
			}
		}
	}

	require.Equal(t, []string{"foo1\n", "foo2\n", "foo3\n", "foo4\n", "foo5\n", "foo6\n", "foo7\n", "foo8\n", "foo9\n", "foo10\n"}, result)

	// reset and get a second run:
	result = make([]string, 0, 10)
	for _, log := range logsSink.AllLogs() {
		for _, rl := range log.ResourceLogs().All() {
			for _, sl := range rl.ScopeLogs().All() {
				for _, lr := range sl.LogRecords().All() {
					result = append(result, lr.Body().Str())
				}
			}
		}
	}

	require.Equal(t, []string{"foo1\n", "foo2\n", "foo3\n", "foo4\n", "foo5\n", "foo6\n", "foo7\n", "foo8\n", "foo9\n", "foo10\n"}, result)
}

func TestRunDisabled(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	r := NewRunner(logsSink, l)
	defer r.Shutdown()
	scheduling, err := r.Run(filepath.Join("testdata", "disabled"))
	require.NoError(t, err)
	require.False(t, scheduling)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 0, logsSink.LogRecordCount())
}

func TestRunDisabledInterval(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	r := NewRunner(logsSink, l)
	defer r.Shutdown()
	scheduling, err := r.Run(filepath.Join("testdata", "disabled_interval"))
	require.NoError(t, err)
	require.False(t, scheduling)
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, logsSink.LogRecordCount(), 0)
}

func TestRunScriptedInputs(t *testing.T) {
	logsSink := &consumertest.LogsSink{}
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	r := NewRunner(logsSink, l)
	defer r.Shutdown()
	scheduling, err := r.Run(filepath.Join("testdata", "script"))
	require.NoError(t, err)
	require.True(t, scheduling)

	require.Eventually(t, func() bool {
		return logsSink.LogRecordCount() == 20
	}, 2*time.Second, 10*time.Millisecond)
}
