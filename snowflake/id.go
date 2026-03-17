package snowflake

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// ID is a 64-bit snowflake identifier.
// Bit layout: [1 sign][41 timestamp][10 node][12 sequence]
type ID int64

// Int64 returns the ID as a raw int64.
func (id ID) Int64() int64 {
	return int64(id)
}

// String returns the decimal string representation of the ID.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Timestamp returns the millisecond offset stored in the ID (relative to the epoch).
func (id ID) Timestamp() int64 {
	return int64(id) >> TimeShift
}

// NodeID returns the 10-bit node/machine identifier embedded in the ID.
func (id ID) NodeID() int64 {
	return (int64(id) >> NodeShift) & MaxNodeID
}

// Sequence returns the 12-bit sequence number embedded in the ID.
func (id ID) Sequence() int64 {
	return int64(id) & MaxSequence
}

// Time converts the embedded timestamp to an absolute time.Time using the given epoch.
func (id ID) Time(epoch time.Time) time.Time {
	ms := id.Timestamp()
	return epoch.Add(time.Duration(ms) * time.Millisecond)
}

// MarshalJSON serializes the ID as a JSON string to avoid JavaScript integer precision loss.
func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(id), 10))
}

// UnmarshalJSON deserializes the ID from a JSON string or number.
func (id *ID) UnmarshalJSON(data []byte) error {
	// Try string first (the expected format)
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("snowflake: invalid ID string %q: %w", s, err)
		}
		*id = ID(v)
		return nil
	}

	// Fall back to number (for interoperability)
	var n int64
	if err := json.Unmarshal(data, &n); err != nil {
		return fmt.Errorf("snowflake: cannot unmarshal %s as ID", string(data))
	}
	*id = ID(n)
	return nil
}
