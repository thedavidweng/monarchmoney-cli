package safety

import (
	"fmt"

	"github.com/monarchmoney-cli/monarch/internal/errors"
)

type OperationTier string

const (
	TierRead         OperationTier = "read"
	TierRemoteAction OperationTier = "remote_action"
	TierMutation     OperationTier = "mutation"
	TierDestructive  OperationTier = "destructive"
)

// Check validates if an operation is allowed based on global flags and tier.
func Check(tier OperationTier, readOnly, dryRun, confirmed bool) error {
	if tier == TierRead {
		return nil
	}

	if readOnly {
		return errors.New(errors.ReadOnlyViolation, "remote writes are blocked in read-only mode", errors.CatSafety, false, nil)
	}

	if dryRun {
		// Dry-run is always allowed for mutations as it makes no remote changes
		return nil
	}

	if !confirmed {
		return errors.New(errors.ConfirmationRequired, fmt.Sprintf("this %s operation requires --confirm to execute", tier), errors.CatSafety, false, nil)
	}

	return nil
}
