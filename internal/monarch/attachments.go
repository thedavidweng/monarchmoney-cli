package monarch

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
)
var newAttachmentRequest = http.NewRequestWithContext

type Attachment struct {
	ID string `json:"id"`
}

func (s *Service) ListTransactionAttachments(ctx context.Context, txID string) ([]Attachment, error) {
	return nil, featureUnavailable("transaction attachments are unavailable in the current Monarch API")
}

func (s *Service) DownloadAttachment(ctx context.Context, url string, w io.Writer) error {
	req, err := newAttachmentRequest(ctx, "GET", url, nil)
	if err != nil {
		return errors.New(errors.InternalError, "failed to create download request", errors.CatInternal, false, err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(errors.NetworkUnreachable, "failed to reach attachment URL", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, "failed to download attachment", errors.CatAPI, false, nil)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

func (s *Service) UploadAttachment(ctx context.Context, txID, path string) error {
	return featureUnavailable("transaction attachment upload is unavailable in the current Monarch API")
}
