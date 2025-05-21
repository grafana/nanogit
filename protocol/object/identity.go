package object

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Identity represents a Git identity (author or committer) in its raw form.
// This matches Git's internal format: "name <email> timestamp timezone"
type Identity struct {
	Name      string
	Email     string
	Timestamp int64
	Timezone  string
}

// ParseIdentity parses a Git identity string in the format "name <email> timestamp timezone"
// and returns an Identity struct.
func ParseIdentity(identity string) (*Identity, error) {
	// Find the last occurrence of '>' which marks the end of the email
	emailEnd := strings.LastIndex(identity, ">")
	if emailEnd == -1 {
		return nil, fmt.Errorf("invalid identity format: %s", identity)
	}

	// Find the start of the email
	emailStart := strings.LastIndex(identity[:emailEnd], "<")
	if emailStart == -1 {
		return nil, fmt.Errorf("invalid identity format: %s", identity)
	}

	// Extract name and email
	name := strings.TrimSpace(identity[:emailStart])
	email := identity[emailStart+1 : emailEnd]

	// Parse timestamp and timezone
	timeStr := strings.TrimSpace(identity[emailEnd+1:])
	parts := strings.Split(timeStr, " ")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid time format: %s", timeStr)
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	return &Identity{
		Name:      name,
		Email:     email,
		Timestamp: timestamp,
		Timezone:  parts[1],
	}, nil
}

// Time returns the time.Time representation of the identity's timestamp and timezone.
func (i *Identity) Time() (time.Time, error) {
	// Parse timezone offset
	if len(i.Timezone) != 5 {
		return time.Time{}, fmt.Errorf("invalid timezone offset format: %s", i.Timezone)
	}

	sign := i.Timezone[0]
	if sign != '+' && sign != '-' {
		return time.Time{}, fmt.Errorf("invalid timezone sign: %c", sign)
	}

	hours, err := strconv.Atoi(i.Timezone[1:3])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hours: %w", err)
	}

	minutes, err := strconv.Atoi(i.Timezone[3:5])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minutes: %w", err)
	}

	// Convert to seconds
	seconds := hours*3600 + minutes*60
	if sign == '-' {
		seconds = -seconds
	}

	loc := time.FixedZone("", seconds)
	return time.Unix(i.Timestamp, 0).In(loc), nil
}
