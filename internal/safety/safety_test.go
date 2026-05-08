package safety

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		tier      OperationTier
		readOnly  bool
		dryRun    bool
		confirmed bool
		wantErr   string
	}{
		{
			name: "Read allowed in read-only",
			tier: TierRead,
			readOnly: true,
			wantErr: "",
		},
		{
			name: "Mutation blocked in read-only",
			tier: TierMutation,
			readOnly: true,
			wantErr: "remote writes are blocked in read-only mode",
		},
		{
			name: "Mutation allowed in dry-run even in read-only? No, blocked if read-only",
			// Wait, my implementation check tier then readOnly then dryRun.
			// Let's check implementing again.
			tier: TierMutation,
			readOnly: true,
			dryRun: true,
			wantErr: "remote writes are blocked in read-only mode",
		},
		{
			name: "Mutation requires confirm",
			tier: TierMutation,
			readOnly: false,
			dryRun: false,
			confirmed: false,
			wantErr: "requires --confirm",
		},
		{
			name: "Mutation allowed with confirm",
			tier: TierMutation,
			readOnly: false,
			dryRun: false,
			confirmed: true,
			wantErr: "",
		},
		{
			name: "Mutation allowed with dry-run",
			tier: TierMutation,
			readOnly: false,
			dryRun: true,
			confirmed: false,
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.tier, tt.readOnly, tt.dryRun, tt.confirmed)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
