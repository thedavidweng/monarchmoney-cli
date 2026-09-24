package safety

import (
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name      string
		tier      OperationTier
		readOnly  bool
		dryRun    bool
		confirmed bool
		wantCode  errors.Code
	}{
		{name: "read allowed in read-only", tier: TierRead, readOnly: true},
		{name: "mutation blocked in read-only", tier: TierMutation, readOnly: true, wantCode: errors.ReadOnlyViolation},
		{name: "read-only precedes dry-run", tier: TierMutation, readOnly: true, dryRun: true, wantCode: errors.ReadOnlyViolation},
		{name: "mutation requires confirm", tier: TierMutation, wantCode: errors.ConfirmationRequired},
		{name: "remote action requires confirm", tier: TierRemoteAction, wantCode: errors.ConfirmationRequired},
		{name: "destructive requires confirm", tier: TierDestructive, wantCode: errors.ConfirmationRequired},
		{name: "mutation allowed with confirm", tier: TierMutation, confirmed: true},
		{name: "mutation allowed with dry-run", tier: TierMutation, dryRun: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Check(tt.tier, tt.readOnly, tt.dryRun, tt.confirmed)
			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("Check() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Check() = nil, want %q", tt.wantCode)
			}
			if err.Code != tt.wantCode {
				t.Fatalf("Check() code = %q, want %q", err.Code, tt.wantCode)
			}
		})
	}
}
