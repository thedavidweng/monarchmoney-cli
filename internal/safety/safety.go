package safety

import (
	"fmt"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)

type OperationTier string

const (
	TierRead         OperationTier = "read"
	TierRemoteAction OperationTier = "remote_action"
	TierMutation     OperationTier = "mutation"
	TierDestructive  OperationTier = "destructive"
)

func Check(tier OperationTier, readOnly, dryRun, confirmed bool) *errors.Error {
	if tier == TierRead {
		return nil
	}

	if readOnly {
		return errors.New(errors.ReadOnlyViolation, "remote writes are blocked in read-only mode", errors.CatSafety, false, nil)
	}

	if dryRun {
		return nil
	}

	if !confirmed {
		return errors.New(errors.ConfirmationRequired, fmt.Sprintf("this %s operation requires --confirm to execute", tier), errors.CatSafety, false, nil)
	}

	return nil
}
