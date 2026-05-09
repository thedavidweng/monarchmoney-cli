package monarch

import "github.com/thedavidweng/monarchmoney-cli/internal/errors"

func featureUnavailable(message string) error {
	return errors.New(errors.FEATURE_UNAVAILABLE, message, errors.CatAPI, false, nil)
}
