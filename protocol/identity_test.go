package protocol

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseIdentity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Identity
		wantErr bool
	}{
		{
			name:  "valid identity",
			input: "John Doe <john@example.com> 1234567890 +0200",
			want: &Identity{
				Name:      "John Doe",
				Email:     "john@example.com",
				Timestamp: 1234567890,
				Timezone:  "+0200",
			},
			wantErr: false,
		},
		{
			name:  "valid identity with spaces in name",
			input: "John A. Doe <john@example.com> 1234567890 -0500",
			want: &Identity{
				Name:      "John A. Doe",
				Email:     "john@example.com",
				Timestamp: 1234567890,
				Timezone:  "-0500",
			},
			wantErr: false,
		},
		{
			name:    "missing email brackets",
			input:   "John Doe john@example.com 1234567890 +0200",
			wantErr: true,
		},
		{
			name:    "invalid timestamp",
			input:   "John Doe <john@example.com> invalid +0200",
			wantErr: true,
		},
		{
			name:    "invalid timezone format",
			input:   "John Doe <john@example.com> 1234567890 +200",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIdentity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareIdentity(got, tt.want) {
				t.Errorf("ParseIdentity() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestIdentity_Time(t *testing.T) {
	tests := []struct {
		name     string
		identity *Identity
		want     time.Time
		wantErr  bool
	}{
		{
			name: "valid timezone +0200",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "+0200",
			},
			want:    time.Unix(1234567890, 0).In(time.FixedZone("", 7200)),
			wantErr: false,
		},
		{
			name: "valid timezone -0500",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "-0500",
			},
			want:    time.Unix(1234567890, 0).In(time.FixedZone("", -18000)),
			wantErr: false,
		},
		{
			name: "invalid timezone format",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "+200",
			},
			wantErr: true,
		},
		{
			name: "invalid timezone sign",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "*0200",
			},
			wantErr: true,
		},
		{
			name: "invalid hours",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "+2500",
			},
			wantErr: true,
		},
		{
			name: "invalid minutes",
			identity: &Identity{
				Timestamp: 1234567890,
				Timezone:  "+0260",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.identity.Time()
			if (err != nil) != tt.wantErr {
				t.Errorf("Identity.Time() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("Identity.Time() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_String(t *testing.T) {
	tests := []struct {
		name     string
		identity Identity
		want     string
	}{
		{
			name: "basic identity",
			identity: Identity{
				Name:      "John Doe",
				Email:     "john@example.com",
				Timestamp: 1234567890,
				Timezone:  "+0000",
			},
			want: "John Doe <john@example.com> 1234567890 +0000",
		},
		{
			name: "identity with special characters",
			identity: Identity{
				Name:      "José García",
				Email:     "jose.garcia@example.com",
				Timestamp: 1234567890,
				Timezone:  "-0700",
			},
			want: "José García <jose.garcia@example.com> 1234567890 -0700",
		},
		{
			name: "identity with empty name",
			identity: Identity{
				Name:      "",
				Email:     "anonymous@example.com",
				Timestamp: 1234567890,
				Timezone:  "+0000",
			},
			want: " <anonymous@example.com> 1234567890 +0000",
		},
		{
			name: "identity with empty email",
			identity: Identity{
				Name:      "John Doe",
				Email:     "",
				Timestamp: 1234567890,
				Timezone:  "+0000",
			},
			want: "John Doe <> 1234567890 +0000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.identity.String()
			require.Equal(t, tt.want, got)
		})
	}
}

// compareIdentity compares two Identity structs for equality
func compareIdentity(a, b *Identity) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Name == b.Name &&
		a.Email == b.Email &&
		a.Timestamp == b.Timestamp &&
		a.Timezone == b.Timezone
}
