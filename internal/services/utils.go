package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

var isoDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TodayYYYYMMDD() string {
	return time.Now().Format("2006-01-02")
}

func ValidateDateYYYYMMDD(date string) bool {
	if !isoDateRegex.MatchString(date) {
		return false
	}
	return true
}

// NowTimestamp returns a UTC timestamp string for CSV storage.
// We keep it as a plain string to avoid time parsing complexity.
func NowTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func NewID(prefix string) string {
	// Small, dependency-free random ID (good enough for MVP).
	b := make([]byte, 6) // 12 hex chars
	_, err := rand.Read(b)
	if err != nil {
		// Extremely unlikely; fall back to time-based ID.
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b))
}

