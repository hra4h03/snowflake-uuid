package snowflake

import (
	"encoding/json"
	"testing"
	"time"
)

func TestIDDecomposition(t *testing.T) {
	// Construct an ID with known fields:
	// timestamp = 1000, nodeID = 42, sequence = 7
	var ts int64 = 1000
	var node int64 = 42
	var seq int64 = 7
	id := ID((ts << TimeShift) | (node << NodeShift) | seq)

	if got := id.Timestamp(); got != ts {
		t.Errorf("Timestamp() = %d, want %d", got, ts)
	}
	if got := id.NodeID(); got != node {
		t.Errorf("NodeID() = %d, want %d", got, node)
	}
	if got := id.Sequence(); got != seq {
		t.Errorf("Sequence() = %d, want %d", got, seq)
	}
}

func TestIDString(t *testing.T) {
	id := ID(123456789)
	if got := id.String(); got != "123456789" {
		t.Errorf("String() = %q, want %q", got, "123456789")
	}
}

func TestIDInt64(t *testing.T) {
	id := ID(42)
	if got := id.Int64(); got != 42 {
		t.Errorf("Int64() = %d, want %d", got, 42)
	}
}

func TestIDTime(t *testing.T) {
	epoch := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var ts int64 = 5000 // 5 seconds after epoch
	id := ID(ts << TimeShift)

	got := id.Time(epoch)
	want := epoch.Add(5 * time.Second)

	if !got.Equal(want) {
		t.Errorf("Time() = %v, want %v", got, want)
	}
}

func TestIDMarshalJSON(t *testing.T) {
	id := ID(458925834012672001)

	data, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	// Should be a quoted string, not a bare number
	want := `"458925834012672001"`
	if string(data) != want {
		t.Errorf("MarshalJSON = %s, want %s", string(data), want)
	}
}

func TestIDUnmarshalJSONString(t *testing.T) {
	data := []byte(`"458925834012672001"`)
	var id ID
	if err := json.Unmarshal(data, &id); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if id.Int64() != 458925834012672001 {
		t.Errorf("UnmarshalJSON = %d, want %d", id.Int64(), 458925834012672001)
	}
}

func TestIDUnmarshalJSONNumber(t *testing.T) {
	data := []byte(`12345`)
	var id ID
	if err := json.Unmarshal(data, &id); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if id.Int64() != 12345 {
		t.Errorf("UnmarshalJSON = %d, want %d", id.Int64(), 12345)
	}
}

func TestIDUnmarshalJSONInvalid(t *testing.T) {
	data := []byte(`"not-a-number"`)
	var id ID
	if err := json.Unmarshal(data, &id); err == nil {
		t.Error("expected error for invalid string, got nil")
	}
}

func TestIDMarshalRoundTrip(t *testing.T) {
	original := ID(9876543210)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if original != decoded {
		t.Errorf("round-trip mismatch: %d != %d", original, decoded)
	}
}

func TestIDDecompositionBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		ts, node, seq int64
	}{
		{"zeros", 0, 0, 0},
		{"max_node", 100, MaxNodeID, 0},
		{"max_seq", 100, 0, MaxSequence},
		{"all_max", (1 << 41) - 1, MaxNodeID, MaxSequence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := ID((tt.ts << TimeShift) | (tt.node << NodeShift) | tt.seq)

			if got := id.Timestamp(); got != tt.ts {
				t.Errorf("Timestamp() = %d, want %d", got, tt.ts)
			}
			if got := id.NodeID(); got != tt.node {
				t.Errorf("NodeID() = %d, want %d", got, tt.node)
			}
			if got := id.Sequence(); got != tt.seq {
				t.Errorf("Sequence() = %d, want %d", got, tt.seq)
			}
		})
	}
}
