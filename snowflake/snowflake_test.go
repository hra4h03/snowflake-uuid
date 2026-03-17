package snowflake

import (
	"sync"
	"testing"
	"time"
)

func TestNewValidNodeIDs(t *testing.T) {
	for _, id := range []int64{0, 1, 512, MaxNodeID} {
		g, err := New(id)
		if err != nil {
			t.Errorf("New(%d) unexpected error: %v", id, err)
		}
		if g.NodeID() != id {
			t.Errorf("NodeID() = %d, want %d", g.NodeID(), id)
		}
	}
}

func TestNewInvalidNodeIDs(t *testing.T) {
	for _, id := range []int64{-1, MaxNodeID + 1, 2048} {
		_, err := New(id)
		if err != ErrInvalidNodeID {
			t.Errorf("New(%d) error = %v, want ErrInvalidNodeID", id, err)
		}
	}
}

func TestBitLayout(t *testing.T) {
	epoch := time.Now().Add(-10 * time.Second)
	var nodeID int64 = 42

	g, err := New(nodeID, WithEpoch(epoch))
	if err != nil {
		t.Fatal(err)
	}

	id, err := g.Generate()
	if err != nil {
		t.Fatal(err)
	}

	// Verify node ID is correctly embedded
	if got := id.NodeID(); got != nodeID {
		t.Errorf("NodeID() = %d, want %d", got, nodeID)
	}

	// Verify timestamp is in a reasonable range (8–12 seconds)
	ts := id.Timestamp()
	if ts < 8000 || ts > 12000 {
		t.Errorf("Timestamp() = %d, want 8000–12000", ts)
	}

	// Verify sequence starts at 0
	if got := id.Sequence(); got != 0 {
		t.Errorf("Sequence() = %d, want 0", got)
	}
}

func TestSignBitAlwaysZero(t *testing.T) {
	g, _ := New(0)

	for i := 0; i < 10000; i++ {
		id, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if id.Int64() < 0 {
			t.Fatalf("generated negative ID: %d", id.Int64())
		}
	}
}

func TestUniqueness(t *testing.T) {
	g, _ := New(1)
	const count = 1_000_000
	seen := make(map[ID]struct{}, count)

	for i := 0; i < count; i++ {
		id, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate ID at iteration %d: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}

func TestMonotonicity(t *testing.T) {
	g, _ := New(1)
	const count = 100_000

	var prev ID
	for i := 0; i < count; i++ {
		id, err := g.Generate()
		if err != nil {
			t.Fatal(err)
		}
		if id <= prev && i > 0 {
			t.Fatalf("non-monotonic at %d: %d <= %d", i, id, prev)
		}
		prev = id
	}
}

func TestConcurrency(t *testing.T) {
	g, _ := New(1)

	const goroutines = 100
	const perGoroutine = 10_000

	ids := make(chan ID, goroutines*perGoroutine)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id, err := g.Generate()
				if err != nil {
					t.Error(err)
					return
				}
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[ID]struct{}, goroutines*perGoroutine)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate concurrent ID: %s", id)
		}
		seen[id] = struct{}{}
	}

	if len(seen) != goroutines*perGoroutine {
		t.Errorf("got %d IDs, want %d", len(seen), goroutines*perGoroutine)
	}
}

func TestSequenceOverflow(t *testing.T) {
	// Use a frozen clock to force sequence overflow
	frozen := time.Now()
	clockMu := sync.Mutex{}
	mockClock := func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return frozen
	}
	advanceClock := func(d time.Duration) {
		clockMu.Lock()
		defer clockMu.Unlock()
		frozen = frozen.Add(d)
	}

	g, _ := New(1, withTimeFunc(mockClock))

	// Generate MaxSequence+1 IDs at the same timestamp
	for i := int64(0); i <= MaxSequence; i++ {
		id, err := g.Generate()
		if err != nil {
			t.Fatalf("Generate() error at seq %d: %v", i, err)
		}
		if id.Sequence() != i {
			t.Fatalf("Sequence() = %d, want %d", id.Sequence(), i)
		}
	}

	// The next Generate() would spin-wait. Advance the clock from another goroutine.
	done := make(chan ID, 1)
	go func() {
		id, err := g.Generate()
		if err != nil {
			t.Error(err)
			return
		}
		done <- id
	}()

	// Let the spin-wait run briefly, then advance the clock
	time.Sleep(1 * time.Millisecond)
	advanceClock(1 * time.Millisecond)

	select {
	case id := <-done:
		if id.Sequence() != 0 {
			t.Errorf("after overflow, Sequence() = %d, want 0", id.Sequence())
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for Generate() after sequence overflow")
	}
}

func TestClockBackwardSmall(t *testing.T) {
	// Clock goes backward by 2ms (within 5ms tolerance) — should recover
	now := time.Now()
	callCount := 0
	mu := sync.Mutex{}

	mockClock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		if callCount == 2 {
			// Second call: 2ms in the past
			return now.Add(-2 * time.Millisecond)
		}
		// All other calls: advance by 1ms each
		return now.Add(time.Duration(callCount) * time.Millisecond)
	}

	g, _ := New(1, withTimeFunc(mockClock))

	// First call succeeds
	if _, err := g.Generate(); err != nil {
		t.Fatalf("first Generate() error: %v", err)
	}

	// Second call: clock went back 2ms, but tolerance is 5ms — should spin-wait and succeed
	if _, err := g.Generate(); err != nil {
		t.Fatalf("second Generate() error (small backward): %v", err)
	}
}

func TestClockBackwardLarge(t *testing.T) {
	// Clock goes backward by 10ms (exceeds 5ms tolerance) — should error
	now := time.Now()
	callCount := 0
	mu := sync.Mutex{}

	mockClock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		if callCount == 1 {
			return now
		}
		// All subsequent calls: 10ms in the past relative to first call
		return now.Add(-10 * time.Millisecond)
	}

	g, _ := New(1, withTimeFunc(mockClock))

	// First call succeeds
	if _, err := g.Generate(); err != nil {
		t.Fatalf("first Generate() error: %v", err)
	}

	// Second call: clock went back 10ms — should error
	_, err := g.Generate()
	if err == nil {
		t.Fatal("expected error for large clock backward, got nil")
	}
}

func TestWithEpoch(t *testing.T) {
	customEpoch := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	g, err := New(1, WithEpoch(customEpoch))
	if err != nil {
		t.Fatal(err)
	}

	id, err := g.Generate()
	if err != nil {
		t.Fatal(err)
	}

	// Timestamp should reflect time since our custom epoch (years ago)
	ts := id.Timestamp()
	expectedMin := time.Since(customEpoch).Milliseconds() - 1000
	if ts < expectedMin {
		t.Errorf("Timestamp() = %d, expected at least %d", ts, expectedMin)
	}
}

func TestDifferentNodesProduceDifferentIDs(t *testing.T) {
	epoch := time.Now()
	g1, _ := New(1, WithEpoch(epoch))
	g2, _ := New(2, WithEpoch(epoch))

	id1, _ := g1.Generate()
	id2, _ := g2.Generate()

	if id1 == id2 {
		t.Error("IDs from different nodes should differ")
	}
	if id1.NodeID() == id2.NodeID() {
		t.Error("NodeIDs should differ")
	}
}

func BenchmarkGenerate(b *testing.B) {
	g, _ := New(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Generate()
	}
}

func BenchmarkGenerateParallel(b *testing.B) {
	g, _ := New(1)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = g.Generate()
		}
	})
}
