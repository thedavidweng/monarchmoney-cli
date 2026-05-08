package monarch

import (
	"context"
	_ "embed"
	"io"
	"net/http"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/transactions/attachments_list.graphql
var GetTransactionAttachmentsQuery string

type Attachment struct {
	ID        string `json:"id"`
	FileName  string `json:"file_name"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

func (s *Service) ListTransactionAttachments(ctx context.Context, txID string) ([]Attachment, error) {
	var resp struct {
		Transaction struct {
			Attachments []struct {
				ID        string `json:"id"`
				FileName  string `json:"fileName"`
				URL       string `json:"url"`
				CreatedAt string `json:"createdAt"`
			} `json:"attachments"`
		} `json:"transaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionAttachments",
		Query:         GetTransactionAttachmentsQuery,
		Variables:     map[string]interface{}{"id": txID},
	}, &resp)

	if err != nil {
		return nil, err
	}

	atts := make([]Attachment, len(resp.Transaction.Attachments))
	for i, a := range resp.Transaction.Attachments {
		atts[i] = Attachment{
			ID:        a.ID,
			FileName:  a.FileName,
			URL:       a.URL,
			CreatedAt: a.CreatedAt,
		}
	}

	return atts, nil
}

func (s *Service) DownloadAttachment(ctx context.Context, url string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
