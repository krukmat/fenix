// Package uuid provides UUID v7 generation.
// UUID v7 is sortable by timestamp (better for database indexes than v4).
package uuid

import (
	"fmt"
	"math/rand"
	"time"
)

// UUID represents a UUID v7 identifier.
type UUID [16]byte

// NewV7 generates a new UUID v7.
// UUID v7 format (as per draft-ietf-uuidrev-rfc4122bis):
// - 48 bits: UNIX timestamp in milliseconds
// - 12 bits: random "sub_ms_seq_hi_and_version"
// - 2 bits: variant
// - 62 bits: random "sub_ms_seq_low"
func NewV7() UUID {
	now := time.Now().UnixMilli()

	var uuid UUID

	// Timestamp (48 bits, ms precision) — bytes 0-5
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Random part (64 bits) — bytes 6-15
	// Sub-ms seq hi (4 bits) + version 0111 (4 bits) = 0x7n
	randomVal := rand.Uint64()
	uuid[6] = 0x70 | byte((randomVal>>56)&0x0f)

	// Variant + random (6 bits variant, 62 bits random)
	// Variant 10xxxxxx in RFC 4122
	uuid[7] = 0x80 | byte((randomVal>>48)&0x3f)
	uuid[8] = byte(randomVal >> 40)
	uuid[9] = byte(randomVal >> 32)
	uuid[10] = byte(randomVal >> 24)
	uuid[11] = byte(randomVal >> 16)
	uuid[12] = byte(randomVal >> 8)
	uuid[13] = byte(randomVal)

	// Final 2 random bytes
	uuid[14] = byte(rand.Intn(256))
	uuid[15] = byte(rand.Intn(256))

	return uuid
}

// String returns the UUID in standard form: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func (u UUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4],
		u[4:6],
		u[6:8],
		u[8:10],
		u[10:16],
	)
}
