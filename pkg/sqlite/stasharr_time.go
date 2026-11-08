package sqlite

import "time"

// ParseStatsTime reads a time stored as text in any of the usual formats.
func ParseStatsTime(s string) (time.Time, bool) {
	for _, layout := range []string{TimestampFormat, time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
