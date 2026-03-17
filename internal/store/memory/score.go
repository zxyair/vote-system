package memory

import (
	"hash/crc32"
	"time"
)

func createdScore(t time.Time, id string) float64 {
	frac := float64(crc32.ChecksumIEEE([]byte(id))%1_000_000) / 1_000_000
	return float64(t.Unix()) + frac
}

