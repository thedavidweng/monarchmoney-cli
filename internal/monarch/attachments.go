package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetTransactionAttachmentsQuery = queries.Get("transactions/attachments_list.graphql")
var GetTransactionAttachmentUploadInfoMutation = queries.Get("transactions/attachment_upload_info.graphql")
var AddTransactionAttachmentMutation = queries.Get("transactions/attachment_add.graphql")
var newAttachmentRequest = http.NewRequestWithContext
var createAttachmentFormFile = func(w *multipart.Writer, field, filename string) (io.Writer, error) {
	return w.CreateFormFile(field, filename)
}
var openAttachmentFile = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

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
	// Step 1: Get upload info
	var infoResp struct {
		GetTransactionAttachmentUploadInfo struct {
			Info struct {
				RequestParams struct {
					Timestamp    int64  `json:"timestamp"`
					Folder       string `json:"folder"`
					Signature    string `json:"signature"`
					APIKey       string `json:"api_key"`
					UploadPreset string `json:"upload_preset"`
				} `json:"requestParams"`
			} `json:"info"`
		} `json:"getTransactionAttachmentUploadInfo"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionAttachmentUploadInfo",
		Query:         GetTransactionAttachmentUploadInfoMutation,
		Variables:     map[string]interface{}{"transactionId": txID},
	}, &infoResp)

	if err != nil {
		return err
	}

	params := infoResp.GetTransactionAttachmentUploadInfo.Info.RequestParams

	// Step 2: Upload to Cloudinary
	file, err := openAttachmentFile(path)
	if err != nil {
		return errors.New(errors.InternalError, "failed to open attachment file", errors.CatInternal, false, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	filename := filepath.Base(path)
	part, err := createAttachmentFormFile(writer, "file", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	writer.WriteField("timestamp", fmt.Sprintf("%d", params.Timestamp))
	writer.WriteField("folder", params.Folder)
	writer.WriteField("signature", params.Signature)
	writer.WriteField("api_key", params.APIKey)
	writer.WriteField("upload_preset", params.UploadPreset)
	writer.Close()

	uploadURL := "https://api.cloudinary.com/v1_1/monarch-money/image/upload/"
	req, err := newAttachmentRequest(ctx, "POST", uploadURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New(errors.NetworkUnreachable, "failed to upload to Cloudinary", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, fmt.Sprintf("Cloudinary upload failed with status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	var cloudResp struct {
		PublicID string `json:"public_id"`
		Format   string `json:"format"`
		Bytes    int    `json:"bytes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cloudResp); err != nil {
		return err
	}

	// Step 3: Link to transaction
	var addResp struct {
		AddTransactionAttachment struct {
			Attachment struct {
				ID string `json:"id"`
			} `json:"attachment"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"addTransactionAttachment"`
	}

	err = s.Client.Do(ctx, &graphql.Request{
		OperationName: "AddTransactionAttachment",
		Query:         AddTransactionAttachmentMutation,
		Variables: map[string]interface{}{
			"input": map[string]interface{}{
				"transactionId": txID,
				"filename":      filename,
				"publicId":      cloudResp.PublicID,
				"extension":     cloudResp.Format,
				"sizeBytes":     cloudResp.Bytes,
			},
		},
	}, &addResp)

	if err != nil {
		return err
	}

	if len(addResp.AddTransactionAttachment.Errors) > 0 {
		return errors.New(errors.APIError, addResp.AddTransactionAttachment.Errors[0].Message, errors.CatAPI, false, nil)
	}

	return nil
}
