package uuid

import (
	"crypto/rand"
	"fmt"
	"time"
)

type UUID [16]byte

func NewV7() UUID {
	now := time.Now().UnixMilli()
	var uuid UUID

	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	randomBytes := randomUUIDBytes()
	uuid[6] = 0x70 | (randomBytes[0] & 0x0f)
	uuid[7] = 0x80 | (randomBytes[1] & 0x3f)
	copy(uuid[8:], randomBytes[2:])

	return uuid
}

func randomUUIDBytes() [10]byte {
	var randomBytes [10]byte
	_, _ = rand.Read(randomBytes[:])
	return randomBytes
}

func (u UUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4],
		u[4:6],
		u[6:8],
		u[8:10],
		u[10:16],
	)
}
