package protocol_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/nanogit/protocol"
)

func TestRefUpdateRequest_Format(t *testing.T) {
	tests := []struct {
		name    string
		req     protocol.RefUpdateRequest
		wantPkt string
		wantErr bool
		errMsg  string
	}{
		{
			name: "create ref",
			req: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
			wantPkt: "0000000000000000000000000000000000000000 1234567890123456789012345678901234567890 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "update ref",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "abcdef0123456789abcdef0123456789abcdef01",
				RefName: "refs/heads/main",
			},
			wantPkt: "1234567890123456789012345678901234567890 abcdef0123456789abcdef0123456789abcdef01 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "delete ref",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/main",
			},
			wantPkt: "1234567890123456789012345678901234567890 0000000000000000000000000000000000000000 refs/heads/main\000report-status-v2 side-band-64k quiet object-format=sha1 agent=nanogit\n0000",
		},
		{
			name: "invalid old ref hash length",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234", // too short
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
			wantErr: true,
			errMsg:  "invalid old ref hash length",
		},
		{
			name: "invalid new ref hash length",
			req: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "1234", // too short
				RefName: "refs/heads/main",
			},
			wantErr: true,
			errMsg:  "invalid new ref hash length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.req.Format()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)

			// Extract the ref line from the packet
			// The packet format is: <length><ref-line>0000<pack-file><flush>
			// Skip the length prefix (4 bytes) and extract just the ref line
			refLine := string(got[4 : len(got)-len(protocol.EmptyPack)-4]) // remove length prefix, pack file, and flush packet
			assert.Equal(t, tt.wantPkt, refLine)

			// Verify the packet structure
			assert.Equal(t, protocol.EmptyPack, got[len(got)-len(protocol.EmptyPack)-4:len(got)-4], "pack file should be present")
			assert.Equal(t, []byte("0000"), got[len(got)-4:], "should end with flush packet")
		})
	}
}

func TestNewCreateRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		newRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "create main branch",
			newRef:  "1234567890123456789012345678901234567890",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "1234567890123456789012345678901234567890",
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "create feature branch",
			newRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  protocol.ZeroHash,
				NewRef:  "0987654321098765432109876543210987654321",
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protocol.NewCreateRefRequest(tt.refName, tt.newRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewUpdateRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		oldRef  string
		newRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "update main branch",
			oldRef:  "1234567890123456789012345678901234567890",
			newRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  "0987654321098765432109876543210987654321",
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "update feature branch",
			oldRef:  "1111111111111111111111111111111111111111",
			newRef:  "2222222222222222222222222222222222222222",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  "1111111111111111111111111111111111111111",
				NewRef:  "2222222222222222222222222222222222222222",
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protocol.NewUpdateRefRequest(tt.oldRef, tt.newRef, tt.refName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewDeleteRefRequest(t *testing.T) {
	tests := []struct {
		name    string
		oldRef  string
		refName string
		want    protocol.RefUpdateRequest
	}{
		{
			name:    "delete main branch",
			oldRef:  "1234567890123456789012345678901234567890",
			refName: "refs/heads/main",
			want: protocol.RefUpdateRequest{
				OldRef:  "1234567890123456789012345678901234567890",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/main",
			},
		},
		{
			name:    "delete feature branch",
			oldRef:  "0987654321098765432109876543210987654321",
			refName: "refs/heads/feature",
			want: protocol.RefUpdateRequest{
				OldRef:  "0987654321098765432109876543210987654321",
				NewRef:  protocol.ZeroHash,
				RefName: "refs/heads/feature",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := protocol.NewDeleteRefRequest(tt.oldRef, tt.refName)
			assert.Equal(t, tt.want, got)
		})
	}
}
