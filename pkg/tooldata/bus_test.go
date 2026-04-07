package tooldata

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolDataBus_Empty(t *testing.T) {
	bus := NewToolDataBus()
	require.NotNil(t, bus)

	_, ok := bus.Get("any")
	assert.False(t, ok)
}

func TestToolDataBus_PutAndGet(t *testing.T) {
	bus := NewToolDataBus()
	data := []byte("hello,world\n1,2")

	bus.Put("csv:abc123", data)

	got, ok := bus.Get("csv:abc123")
	require.True(t, ok)
	assert.Equal(t, data, got)
}

func TestToolDataBus_Overwrite(t *testing.T) {
	bus := NewToolDataBus()
	bus.Put("ref", []byte("first"))
	bus.Put("ref", []byte("second"))

	got, ok := bus.Get("ref")
	require.True(t, ok)
	assert.Equal(t, []byte("second"), got)
}

func TestToolDataBus_Delete(t *testing.T) {
	bus := NewToolDataBus()
	bus.Put("ref", []byte("data"))
	bus.Delete("ref")

	_, ok := bus.Get("ref")
	assert.False(t, ok)
}

func TestToolDataBus_DeleteNonExistent(t *testing.T) {
	bus := NewToolDataBus()
	// Should not panic
	bus.Delete("does_not_exist")
}

func TestToolDataBus_MultipleRefs(t *testing.T) {
	bus := NewToolDataBus()
	bus.Put("csv:ref1", []byte("file1"))
	bus.Put("csv:ref2", []byte("file2"))
	bus.Put("csv:ref3", []byte("file3"))

	d1, ok1 := bus.Get("csv:ref1")
	d2, ok2 := bus.Get("csv:ref2")
	d3, ok3 := bus.Get("csv:ref3")

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)
	assert.Equal(t, []byte("file1"), d1)
	assert.Equal(t, []byte("file2"), d2)
	assert.Equal(t, []byte("file3"), d3)
}

func TestWithBus_AndBusFromContext(t *testing.T) {
	bus := NewToolDataBus()
	bus.Put("csv:test", []byte("content"))

	ctx := WithBus(context.Background(), bus)

	retrieved := BusFromContext(ctx)
	require.NotNil(t, retrieved)

	data, ok := retrieved.Get("csv:test")
	require.True(t, ok)
	assert.Equal(t, []byte("content"), data)
}

func TestBusFromContext_NoBus(t *testing.T) {
	retrieved := BusFromContext(context.Background())
	assert.Nil(t, retrieved)
}

func TestToolDataBus_ConcurrentAccess(t *testing.T) {
	bus := NewToolDataBus()

	const workers = 100

	type result struct {
		ref  string
		want []byte
		got  []byte
		ok   bool
	}

	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan result, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			ref := fmt.Sprintf("ref-%d", i)
			want := []byte(fmt.Sprintf("data-%d", i))
			bus.Put(ref, want)

			got, ok := bus.Get(ref)
			results <- result{ref: ref, want: want, got: got, ok: ok}
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	for res := range results {
		assert.True(t, res.ok, "expected to retrieve %s", res.ref)
		assert.Equal(t, res.want, res.got, "unexpected payload for %s", res.ref)
	}

	for i := 0; i < workers; i++ {
		ref := fmt.Sprintf("ref-%d", i)
		want := []byte(fmt.Sprintf("data-%d", i))

		got, ok := bus.Get(ref)
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
}
