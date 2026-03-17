package snowflake

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	NodeBits     = 10
	SequenceBits = 12

	MaxNodeID   int64 = (1 << NodeBits) - 1   // 1023
	MaxSequence int64 = (1 << SequenceBits) - 1 // 4095

	TimeShift = NodeBits + SequenceBits // 22
	NodeShift = SequenceBits            // 12
)

// DefaultEpoch is 2024-01-01 00:00:00 UTC.
// With 41-bit millisecond timestamps this gives ~69.7 years of usable range (until ~2093).
var DefaultEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

var (
	ErrInvalidNodeID  = errors.New("snowflake: node ID must be between 0 and 1023")
	ErrClockBackward  = errors.New("snowflake: clock moved backward")
	ErrTimestampLimit = errors.New("snowflake: timestamp exceeds 41-bit limit")
)

// Generator produces unique snowflake IDs. It is safe for concurrent use.
type Generator struct {
	mu       sync.Mutex
	epoch    time.Time
	nodeID   int64
	lastTime int64
	sequence int64
	timeFunc func() time.Time
}

// Option configures a Generator.
type Option func(*Generator)

// WithEpoch sets a custom epoch for timestamp calculation.
func WithEpoch(epoch time.Time) Option {
	return func(g *Generator) {
		g.epoch = epoch
	}
}

// withTimeFunc injects a custom clock source (for testing).
func withTimeFunc(fn func() time.Time) Option {
	return func(g *Generator) {
		g.timeFunc = fn
	}
}

// New creates a Generator for the given node ID (0–1023).
func New(nodeID int64, opts ...Option) (*Generator, error) {
	if nodeID < 0 || nodeID > MaxNodeID {
		return nil, ErrInvalidNodeID
	}

	g := &Generator{
		epoch:    DefaultEpoch,
		nodeID:   nodeID,
		timeFunc: time.Now,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g, nil
}

// Generate produces the next unique snowflake ID.
// It blocks briefly if the sequence overflows within a millisecond or if a
// minor clock-backward event is detected (up to 5 ms tolerance).
func (g *Generator) Generate() (ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.elapsed()

	// Handle clock moving backward
	if now < g.lastTime {
		delta := g.lastTime - now
		if delta <= 5 {
			// Tolerate small backward drift: spin-wait
			for now < g.lastTime {
				now = g.elapsed()
			}
		} else {
			return 0, fmt.Errorf("%w: moved back %d ms", ErrClockBackward, delta)
		}
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & MaxSequence
		if g.sequence == 0 {
			// Sequence exhausted in this millisecond — wait for the next one
			for now <= g.lastTime {
				now = g.elapsed()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	// Guard against 41-bit overflow
	if now>>41 != 0 {
		return 0, ErrTimestampLimit
	}

	id := (now << TimeShift) | (g.nodeID << NodeShift) | g.sequence
	return ID(id), nil
}

// NodeID returns the node ID assigned to this generator.
func (g *Generator) NodeID() int64 {
	return g.nodeID
}

// elapsed returns milliseconds since the configured epoch.
func (g *Generator) elapsed() int64 {
	return g.timeFunc().Sub(g.epoch).Milliseconds()
}
