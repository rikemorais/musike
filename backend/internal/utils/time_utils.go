package utils

import (
	"fmt"
	"time"
)

// GetLocalTime returns current time in system's local timezone
func GetLocalTime() time.Time {
	return time.Now()
}

// ParseSpotifyTimestampToLocal parses Spotify timestamp format and converts to local timezone
func ParseSpotifyTimestampToLocal(timeStr string) (time.Time, error) {
	// Try RFC3339 first (API format)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Local(), nil
	}
	
	// Try Spotify Extended History format
	if t, err := time.Parse("2006-01-02T15:04:05Z", timeStr); err == nil {
		return t.Local(), nil
	}
	
	// Try without timezone info (assume UTC and convert to local)
	if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
		return t.UTC().Local(), nil
	}
	
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timeStr)
}

// GetStartDateForTimeFilter returns start date based on time filter
func GetStartDateForTimeFilter(timeFilter string) time.Time {
	now := GetLocalTime()

	switch timeFilter {
	case "day":
		return now.AddDate(0, 0, -1)
	case "week":
		return now.AddDate(0, 0, -7)
	case "month":
		return now.AddDate(0, -1, 0)
	case "quarter":
		return now.AddDate(0, -3, 0)
	case "semester":
		return now.AddDate(0, -6, 0)
	case "year":
		return now.AddDate(-1, 0, 0)
	case "alltime":
		return time.Time{} // Data zero = sem filtro
	// Manter compatibilidade com filtros antigos
	case "6months":
		return now.AddDate(0, -6, 0)
	case "1year":
		return now.AddDate(-1, 0, 0)
	default:
		return time.Time{} // Default todo o histórico
	}
}