package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type fakeCache struct {
	err  error
	data map[string][]byte
}

func (f *fakeCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *fakeCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.data[key] = value
	return nil
}

func (f *fakeCache) Delete(_ context.Context, key string) error {
	if f.err != nil {
		return f.err
	}
	delete(f.data, key)
	return nil
}

func (f *fakeCache) GetDel(_ context.Context, key string) ([]byte, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.data[key]
	delete(f.data, key)
	return v, ok, nil
}

func TestDegradingSwallowsErrorsAndCounts(t *testing.T) {
	reg := prometheus.NewRegistry()
	inner := &fakeCache{err: errors.New("connection refused")}
	d := NewDegrading(inner, reg)
	ctx := context.Background()

	v, found, err := d.Get(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, v)

	v, found, err = d.GetDel(ctx, "k")
	require.NoError(t, err)
	require.False(t, found)
	require.Nil(t, v)

	require.NoError(t, d.Set(ctx, "k", []byte("v"), time.Minute))
	require.NoError(t, d.Delete(ctx, "k"))

	require.False(t, d.Healthy())
	require.Equal(t, float64(4), testutil.ToFloat64(d.errors))
}

func TestDegradingPassesThroughWhenHealthy(t *testing.T) {
	inner := &fakeCache{data: map[string][]byte{}}
	d := NewDegrading(inner, prometheus.NewRegistry())
	ctx := context.Background()

	require.NoError(t, d.Set(ctx, "k", []byte("v"), time.Minute))
	v, found, err := d.Get(ctx, "k")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []byte("v"), v)
	require.True(t, d.Healthy())
}

func TestDegradingIgnoresCanceledContext(t *testing.T) {
	d := NewDegrading(&fakeCache{err: context.Canceled}, prometheus.NewRegistry())

	_, found, err := d.Get(context.Background(), "k")
	require.NoError(t, err)
	require.False(t, found)
	require.True(t, d.Healthy())
	require.Equal(t, float64(0), testutil.ToFloat64(d.errors))
}
