package tooldata

import (
	"context"
	"sync"
)

type contextKey int

const busKey contextKey = iota

// ToolDataBus is a request-scoped, thread-safe store for large binary payloads
// that must reach tool handlers without transiting through the LLM context.
// It is injected into context.Context via WithBus and lives for the duration
// of a single RunStreaming call.
type ToolDataBus struct {
	mu      sync.RWMutex
	entries map[string][]byte
}

// NewToolDataBus creates an empty ToolDataBus.
func NewToolDataBus() *ToolDataBus {
	return &ToolDataBus{entries: make(map[string][]byte)}
}

// WithBus returns a new context carrying the given bus.
func WithBus(ctx context.Context, bus *ToolDataBus) context.Context {
	return context.WithValue(ctx, busKey, bus)
}

// BusFromContext extracts the ToolDataBus from ctx.
// Returns nil if no bus is present — callers must handle nil.
func BusFromContext(ctx context.Context) *ToolDataBus {
	v, _ := ctx.Value(busKey).(*ToolDataBus)
	return v
}

// Put stores data under ref. Overwrites any existing entry.
func (b *ToolDataBus) Put(ref string, data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[ref] = data
}

// Get retrieves data by ref. Returns (nil, false) if not found.
func (b *ToolDataBus) Get(ref string) ([]byte, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	d, ok := b.entries[ref]
	return d, ok
}

// Delete removes a ref from the bus (call after consumption to free memory).
func (b *ToolDataBus) Delete(ref string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, ref)
}
