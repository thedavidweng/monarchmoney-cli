package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	GetAttachmentUploadInfoMutation  = queries.Get("transactions/get_attachment_upload_info.graphql")
	AddTransactionAttachmentMutation = queries.Get("transactions/add_attachment.graphql")
	newAttachmentRequest             = http.NewRequestWithContext
	attachmentUploadURL              = "https://api.cloudinary.com/v1_1/monarch-money/image/upload/"
	attachmentUploadClient           = &http.Client{Timeout: 60 * time.Second}
	attachmentDownloadClient         = &http.Client{Timeout: 30 * time.Second}
)

type Attachment struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Extension string `json:"extension"`
	URL       string `json:"url"`
	SizeBytes int    `json:"size_bytes"`
}

func (s *Service) ListTransactionAttachments(ctx context.Context, txID string) ([]Attachment, error) {
	var resp struct {
		GetTransaction struct {
			Attachments []struct {
				ID               string `json:"id"`
				Extension        string `json:"extension"`
				Filename         string `json:"filename"`
				OriginalAssetUrl string `json:"originalAssetUrl"`
				SizeBytes        int    `json:"sizeBytes"`
			} `json:"attachments"`
		} `json:"getTransaction"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetTransactionDrawer",
		Query:         GetTransactionQuery,
		Variables:     map[string]any{"id": txID},
	}, &resp)
	if err != nil {
		return nil, err
	}

	attachments := make([]Attachment, len(resp.GetTransaction.Attachments))
	for i, a := range resp.GetTransaction.Attachments {
		attachments[i] = Attachment{
			ID:        a.ID,
			Filename:  a.Filename,
			Extension: a.Extension,
			URL:       a.OriginalAssetUrl,
			SizeBytes: a.SizeBytes,
		}
	}
	return attachments, nil
}

func (s *Service) DownloadAttachment(ctx context.Context, url string, w io.Writer) error {
	req, err := newAttachmentRequest(ctx, "GET", url, nil)
	if err != nil {
		return errors.New(errors.InternalError, "failed to create download request", errors.CatInternal, false, err)
	}

	client := attachmentDownloadClient
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

type attachmentUploadParams struct {
	Timestamp    json.RawMessage `json:"timestamp"`
	Folder       string          `json:"folder"`
	Signature    string          `json:"signature"`
	APIKey       string          `json:"api_key"`
	UploadPreset string          `json:"upload_preset"`
}

func (s *Service) UploadAttachment(ctx context.Context, txID, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return errors.New(errors.InternalError, "failed to read attachment file", errors.CatInternal, false, err)
	}
	filename := filepath.Base(path)

	var infoResp struct {
		GetTransactionAttachmentUploadInfo struct {
			Info struct {
				RequestParams attachmentUploadParams `json:"requestParams"`
			} `json:"info"`
		} `json:"getTransactionAttachmentUploadInfo"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_GetTransactionAttachmentUploadInfo",
		Query:         GetAttachmentUploadInfoMutation,
		Variables:     map[string]any{"transactionId": txID},
	}, &infoResp)
	if err != nil {
		return err
	}

	result, err := s.postAttachmentToCloudinary(ctx, filename, content, &infoResp.GetTransactionAttachmentUploadInfo.Info.RequestParams)
	if err != nil {
		return err
	}

	var addResp struct {
		AddTransactionAttachment struct {
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"addTransactionAttachment"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_AddTransactionAttachment",
		Query:         AddTransactionAttachmentMutation,
		Variables: map[string]any{"input": map[string]any{
			"extension":     result.Format,
			"transactionId": txID,
			"filename":      filename,
			"publicId":      result.PublicID,
			"sizeBytes":     result.Bytes,
		}},
	}, &addResp)
	if err != nil {
		return err
	}
	if addResp.AddTransactionAttachment.Errors != nil && addResp.AddTransactionAttachment.Errors.Message != "" {
		return errors.New(errors.APIError, addResp.AddTransactionAttachment.Errors.Message, errors.CatAPI, false, nil)
	}
	return nil
}

type cloudinaryUploadResult struct {
	PublicID string `json:"public_id"`
	Format   string `json:"format"`
	Bytes    int    `json:"bytes"`
}

func (s *Service) postAttachmentToCloudinary(ctx context.Context, filename string, content []byte, params *attachmentUploadParams) (*cloudinaryUploadResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writeCloudinaryForm(writer, filename, content, params); err != nil {
		return nil, err
	}

	req, err := newAttachmentRequest(ctx, "POST", attachmentUploadURL, body)
	if err != nil {
		return nil, errors.New(errors.InternalError, "failed to create upload request", errors.CatInternal, false, err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", graphql.UserAgent())

	resp, err := attachmentUploadClient.Do(req)
	if err != nil {
		return nil, errors.New(errors.NetworkUnreachable, "failed to reach attachment upload endpoint", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(errors.APIError, fmt.Sprintf("attachment upload failed with status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	var result cloudinaryUploadResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.PublicID == "" {
		return nil, errors.New(errors.APISchemaChanged, "attachment upload response missing public_id", errors.CatAPI, false, err)
	}
	return &result, nil
}

func escapeFormQuotes(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

func writeCloudinaryForm(writer *multipart.Writer, filename string, content []byte, params *attachmentUploadParams) error {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, escapeFormQuotes(filename)))
	if contentType := mime.TypeByExtension(filepath.Ext(filename)); contentType != "" {
		header.Set("Content-Type", contentType)
	} else {
		header.Set("Content-Type", "application/octet-stream")
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return errors.New(errors.InternalError, "failed to create upload form file", errors.CatInternal, false, err)
	}
	if _, err := part.Write(content); err != nil {
		return errors.New(errors.InternalError, "failed to write upload file content", errors.CatInternal, false, err)
	}

	fields := map[string]string{
		"timestamp":     string(params.Timestamp),
		"folder":        params.Folder,
		"signature":     params.Signature,
		"api_key":       params.APIKey,
		"upload_preset": params.UploadPreset,
	}
	keys := []string{"timestamp", "folder", "signature", "api_key", "upload_preset"}
	for _, key := range keys {
		if err := writer.WriteField(key, fields[key]); err != nil {
			return errors.New(errors.InternalError, "failed to write upload form field", errors.CatInternal, false, err)
		}
	}
	return writer.Close()
}
