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
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var (
	CreateBulkRetailSyncMutation  = queries.Get("receipts/create_bulk_retail_sync.graphql")
	StartRetailSyncMutation       = queries.Get("receipts/start_retail_sync.graphql")
	ListReceiptsQuery             = queries.Get("receipts/list.graphql")
	GetReceiptQuery               = queries.Get("receipts/show.graphql")
	DeleteReceiptMutation         = queries.Get("receipts/delete.graphql")
	MatchReceiptMutation          = queries.Get("receipts/match.graphql")
	UnmatchReceiptMutation        = queries.Get("receipts/unmatch.graphql")
	UpdateReceiptMutation         = queries.Get("receipts/update.graphql")
	GetReceiptSettingsQuery       = queries.Get("receipts/settings.graphql")
	UpdateReceiptSettingsMutation = queries.Get("receipts/update_settings.graphql")
	newRetailSyncRequest          = http.NewRequestWithContext
	retailSyncFilesURL            = "https://api.monarch.com/retail-sync/%s/files"
	retailSyncUploadClient        = &http.Client{Timeout: 120 * time.Second}
	newRetailSyncFormFile         = func(w *multipart.Writer, field, filename, contentType string) (io.Writer, error) {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, escapeFormQuotes(filename)))
		header.Set("Content-Type", contentType)
		return w.CreatePart(header)
	}
)

type ReceiptSync struct {
	ID        string `json:"id"`
	Vendor    string `json:"vendor"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at"`
}

type ReceiptAttachment struct {
	ID               string `json:"id"`
	StorageID        string `json:"storage_id,omitempty"`
	Filename         string `json:"filename,omitempty"`
	Extension        string `json:"extension,omitempty"`
	SizeBytes        int    `json:"size_bytes,omitempty"`
	OriginalAssetURL string `json:"original_asset_url,omitempty"`
	ThumbnailURL     string `json:"thumbnail_url,omitempty"`
}

type ReceiptLineItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Quantity     *int     `json:"quantity,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	Total        *float64 `json:"total,omitempty"`
	Associated   *bool    `json:"is_associated_to_transaction,omitempty"`
	CategoryID   string   `json:"category_id,omitempty"`
	CategoryName string   `json:"category_name,omitempty"`
}

type ReceiptLinkedTransaction struct {
	ID        string `json:"id"`
	IsManual  *bool  `json:"is_manual,omitempty"`
	HasSplits *bool  `json:"has_splits,omitempty"`
	Merchant  string `json:"merchant,omitempty"`
}

type ReceiptTransaction struct {
	ID            string                    `json:"id"`
	Date          string                    `json:"date,omitempty"`
	Total         *float64                  `json:"total,omitempty"`
	Type          string                    `json:"type,omitempty"`
	UpdateSkipped *bool                     `json:"update_skipped,omitempty"`
	Linked        *ReceiptLinkedTransaction `json:"transaction,omitempty"`
}

type ReceiptOrder struct {
	ID               string               `json:"id"`
	MerchantName     string               `json:"merchant_name,omitempty"`
	Date             string               `json:"date,omitempty"`
	TotalForProducts *float64             `json:"total_for_products,omitempty"`
	Tax              *float64             `json:"tax,omitempty"`
	Tip              *float64             `json:"tip,omitempty"`
	GrandTotal       *float64             `json:"grand_total,omitempty"`
	DisplayStatus    string               `json:"display_status,omitempty"`
	LineItems        []ReceiptLineItem    `json:"line_items,omitempty"`
	Transactions     []ReceiptTransaction `json:"transactions,omitempty"`
}

type Receipt struct {
	ID          string              `json:"id"`
	Vendor      string              `json:"vendor"`
	Status      string              `json:"status"`
	StartedAt   string              `json:"started_at,omitempty"`
	EndedAt     string              `json:"ended_at,omitempty"`
	CreatedAt   string              `json:"created_at,omitempty"`
	UpdatedAt   string              `json:"updated_at,omitempty"`
	Orders      []ReceiptOrder      `json:"orders,omitempty"`
	Attachments []ReceiptAttachment `json:"attachments,omitempty"`
}

func (r *Receipt) IsMatched() bool {
	for i := range r.Orders {
		for j := range r.Orders[i].Transactions {
			if r.Orders[i].Transactions[j].Linked != nil {
				return true
			}
		}
	}
	return false
}

func (r *Receipt) MatchedTransactionIDs() []string {
	var ids []string
	for i := range r.Orders {
		for j := range r.Orders[i].Transactions {
			if r.Orders[i].Transactions[j].Linked != nil {
				ids = append(ids, r.Orders[i].Transactions[j].Linked.ID)
			}
		}
	}
	return ids
}

type ReceiptSettings struct {
	Vendor                 string `json:"vendor"`
	AutoCategorizeAndSplit *bool  `json:"auto_categorize_and_split,omitempty"`
	UpdateTransactionNotes *bool  `json:"update_transaction_notes,omitempty"`
}

type ReceiptUpdate struct {
	MerchantName   *string
	Date           *string
	TotalBeforeTax *float64
	Tax            *float64
	Tip            *float64
	GrandTotal     *float64
}

func (s *Service) UploadReceiptToInbox(ctx context.Context, path string) (*ReceiptSync, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New(errors.InternalError, "failed to read receipt file", errors.CatInternal, false, err)
	}
	filename := filepath.Base(path)
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	var createResp struct {
		CreateBulkRetailSync struct {
			RetailSyncs []struct {
				ID     string `json:"id"`
				Vendor string `json:"vendor"`
				Status string `json:"status"`
			} `json:"retailSyncs"`
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"createBulkRetailSync"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_CreateBulkRetailSync",
		Query:         CreateBulkRetailSyncMutation,
		Variables:     map[string]any{"input": map[string]any{"count": 1}},
	}, &createResp)
	if err != nil {
		return nil, err
	}
	if len(createResp.CreateBulkRetailSync.RetailSyncs) == 0 || createResp.CreateBulkRetailSync.Errors != nil {
		msg := "failed to create retail sync session"
		if createResp.CreateBulkRetailSync.Errors != nil {
			msg = createResp.CreateBulkRetailSync.Errors.Message
		}
		return nil, errors.New(errors.APIError, msg, errors.CatAPI, false, nil)
	}
	syncID := createResp.CreateBulkRetailSync.RetailSyncs[0].ID

	if err := s.postReceiptFile(ctx, syncID, filename, contentType, content); err != nil {
		return nil, err
	}

	var startResp struct {
		StartRetailSync struct {
			RetailSync *struct {
				ID        string `json:"id"`
				Vendor    string `json:"vendor"`
				Status    string `json:"status"`
				StartedAt string `json:"startedAt"`
				EndedAt   string `json:"endedAt"`
			} `json:"retailSync"`
			Errors *struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"startRetailSync"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_StartRetailSync",
		Query:         StartRetailSyncMutation,
		Variables:     map[string]any{"syncId": syncID},
	}, &startResp)
	if err != nil {
		return nil, err
	}
	if startResp.StartRetailSync.RetailSync == nil || startResp.StartRetailSync.Errors != nil {
		msg := "failed to start retail sync"
		if startResp.StartRetailSync.Errors != nil {
			msg = startResp.StartRetailSync.Errors.Message
		}
		return nil, errors.New(errors.APIError, msg, errors.CatAPI, false, nil)
	}

	started := startResp.StartRetailSync.RetailSync
	return &ReceiptSync{
		ID:        started.ID,
		Vendor:    started.Vendor,
		Status:    started.Status,
		StartedAt: started.StartedAt,
		EndedAt:   started.EndedAt,
	}, nil
}

func (s *Service) postReceiptFile(ctx context.Context, syncID, filename, contentType string, content []byte) error {
	metadata, err := json.Marshal(map[string]string{
		"orderId":     uuid.NewString(),
		"vendor":      "user_import",
		"payloadType": "order",
		"contentType": contentType,
	})
	if err != nil {
		return errors.New(errors.InternalError, "failed to encode receipt metadata", errors.CatInternal, false, err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("payloads_count", "1"); err != nil {
		return errors.New(errors.InternalError, "failed to write receipt form field", errors.CatInternal, false, err)
	}
	if err := writer.WriteField("metadata_0", string(metadata)); err != nil {
		return errors.New(errors.InternalError, "failed to write receipt form field", errors.CatInternal, false, err)
	}
	part, err := newRetailSyncFormFile(writer, "payload_0", filename, contentType)
	if err != nil {
		return err
	}
	if _, err := part.Write(content); err != nil {
		return errors.New(errors.InternalError, "failed to write receipt content", errors.CatInternal, false, err)
	}
	if err := writer.Close(); err != nil {
		return errors.New(errors.InternalError, "failed to finalize receipt upload body", errors.CatInternal, false, err)
	}

	req, err := newRetailSyncRequest(ctx, "POST", fmt.Sprintf(retailSyncFilesURL, syncID), body)
	if err != nil {
		return errors.New(errors.InternalError, "failed to create receipt upload request", errors.CatInternal, false, err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Client-Platform", "web")
	req.Header.Set("User-Agent", graphql.UserAgent())
	if token := s.Client.TokenValue(); token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}

	resp, err := retailSyncUploadClient.Do(req)
	if err != nil {
		return errors.New(errors.NetworkUnreachable, "failed to reach retail sync endpoint", errors.CatNetwork, true, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, fmt.Sprintf("receipt upload failed with status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}
	return nil
}

type receiptPayloadError struct {
	Message     string `json:"message"`
	Code        string `json:"code"`
	FieldErrors []struct {
		Field    string   `json:"field"`
		Messages []string `json:"messages"`
	} `json:"fieldErrors"`
}

type receiptPayloadErrors struct {
	Items []receiptPayloadError
}

func (e *receiptPayloadErrors) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if string(trimmed) == "null" {
		return nil
	}
	if bytes.HasPrefix(trimmed, []byte("[")) {
		var multi []receiptPayloadError
		if err := json.Unmarshal(data, &multi); err != nil {
			return err
		}
		e.Items = multi
		return nil
	}
	var single receiptPayloadError
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	if single.Message != "" || single.Code != "" || len(single.FieldErrors) > 0 {
		e.Items = []receiptPayloadError{single}
	}
	return nil
}

func (e receiptPayloadErrors) toError(fallback string) *errors.Error {
	for _, item := range e.Items {
		if item.Message != "" {
			return errors.New(errors.APIError, item.Message, errors.CatAPI, false, nil)
		}
		for _, fe := range item.FieldErrors {
			for _, msg := range fe.Messages {
				return errors.New(errors.APIError, fe.Field+": "+msg, errors.CatAPI, false, nil)
			}
		}
	}
	if len(e.Items) > 0 {
		return errors.New(errors.APIError, fallback, errors.CatAPI, false, nil)
	}
	return nil
}

type rawReceipt struct {
	ID        string `json:"id"`
	Vendor    string `json:"vendor"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Orders    []struct {
		ID               string   `json:"id"`
		MerchantName     string   `json:"merchantName"`
		Date             string   `json:"date"`
		TotalForProducts *float64 `json:"totalForProducts"`
		Tax              *float64 `json:"tax"`
		Tip              *float64 `json:"tip"`
		GrandTotal       *float64 `json:"grandTotal"`
		DisplayStatus    string   `json:"displayStatus"`
		RetailLineItems  []struct {
			ID                              string   `json:"id"`
			Title                           string   `json:"title"`
			Quantity                        *int     `json:"quantity"`
			Price                           *float64 `json:"price"`
			Total                           *float64 `json:"total"`
			IsAssociatedToRetailTransaction *bool    `json:"isAssociatedToRetailTransaction"`
			Category                        *struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"category"`
		} `json:"retailLineItems"`
		RetailTransactions []struct {
			ID                       string   `json:"id"`
			Date                     string   `json:"date"`
			Total                    *float64 `json:"total"`
			TransactionType          string   `json:"transactionType"`
			TransactionUpdateSkipped *bool    `json:"transactionUpdateSkipped"`
			Transaction              *struct {
				ID                   string `json:"id"`
				IsManual             *bool  `json:"isManual"`
				HasSplitTransactions *bool  `json:"hasSplitTransactions"`
				Merchant             *struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"transaction"`
		} `json:"retailTransactions"`
	} `json:"orders"`
	Attachments []struct {
		ID               string `json:"id"`
		StorageID        string `json:"storageId"`
		Filename         string `json:"filename"`
		Extension        string `json:"extension"`
		SizeBytes        int    `json:"sizeBytes"`
		OriginalAssetURL string `json:"originalAssetUrl"`
		ThumbnailURL     string `json:"thumbnailUrl"`
	} `json:"attachments"`
}

func toReceipt(r *rawReceipt) *Receipt {
	out := &Receipt{
		ID:        r.ID,
		Vendor:    r.Vendor,
		Status:    r.Status,
		StartedAt: r.StartedAt,
		EndedAt:   r.EndedAt,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	for i := range r.Orders {
		o := &r.Orders[i]
		order := ReceiptOrder{
			ID:               o.ID,
			MerchantName:     o.MerchantName,
			Date:             o.Date,
			TotalForProducts: o.TotalForProducts,
			Tax:              o.Tax,
			Tip:              o.Tip,
			GrandTotal:       o.GrandTotal,
			DisplayStatus:    o.DisplayStatus,
		}
		for _, li := range o.RetailLineItems {
			item := ReceiptLineItem{
				ID:         li.ID,
				Title:      li.Title,
				Quantity:   li.Quantity,
				Price:      li.Price,
				Total:      li.Total,
				Associated: li.IsAssociatedToRetailTransaction,
			}
			if li.Category != nil {
				item.CategoryID = li.Category.ID
				item.CategoryName = li.Category.Name
			}
			order.LineItems = append(order.LineItems, item)
		}
		for _, rt := range o.RetailTransactions {
			txn := ReceiptTransaction{
				ID:            rt.ID,
				Date:          rt.Date,
				Total:         rt.Total,
				Type:          rt.TransactionType,
				UpdateSkipped: rt.TransactionUpdateSkipped,
			}
			if rt.Transaction != nil {
				linked := &ReceiptLinkedTransaction{
					ID:        rt.Transaction.ID,
					IsManual:  rt.Transaction.IsManual,
					HasSplits: rt.Transaction.HasSplitTransactions,
				}
				if rt.Transaction.Merchant != nil {
					linked.Merchant = rt.Transaction.Merchant.Name
				}
				txn.Linked = linked
			}
			order.Transactions = append(order.Transactions, txn)
		}
		out.Orders = append(out.Orders, order)
	}
	for _, a := range r.Attachments {
		out.Attachments = append(out.Attachments, ReceiptAttachment{
			ID:               a.ID,
			StorageID:        a.StorageID,
			Filename:         a.Filename,
			Extension:        a.Extension,
			SizeBytes:        a.SizeBytes,
			OriginalAssetURL: a.OriginalAssetURL,
			ThumbnailURL:     a.ThumbnailURL,
		})
	}
	return out
}

func (s *Service) listReceiptsPage(ctx context.Context, status, vendor string, limit, offset int) ([]*Receipt, int, error) {
	var resp struct {
		RetailSyncsWithTotal struct {
			TotalCount int           `json:"totalCount"`
			Results    []*rawReceipt `json:"results"`
		} `json:"retailSyncsWithTotal"`
	}
	filters := map[string]any{}
	if status != "" {
		filters["status"] = status
	}
	if vendor != "" {
		filters["vendor"] = vendor
	}
	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_RetailSyncsQueryWithTotal",
		Query:         ListReceiptsQuery,
		Variables: map[string]any{
			"filters":           filters,
			"offset":            offset,
			"limit":             limit,
			"includeTotalCount": true,
		},
	}, &resp)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*Receipt, 0, len(resp.RetailSyncsWithTotal.Results))
	for _, r := range resp.RetailSyncsWithTotal.Results {
		if r != nil {
			out = append(out, toReceipt(r))
		}
	}
	return out, resp.RetailSyncsWithTotal.TotalCount, nil
}

func (s *Service) ListReceipts(ctx context.Context, status, vendor string, limit, offset int, matchedOnly *bool) ([]*Receipt, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	if vendor != "" && matchedOnly == nil {
		return s.listReceiptsPage(ctx, status, vendor, limit, offset)
	}
	vendors := []string{vendor}
	if vendor == "" {
		vendors = []string{"user_import", "email_import"}
	}
	merged := []*Receipt{}
	total := 0
	for _, v := range vendors {
		receipts, vendorTotal, err := s.listAllReceiptsPages(ctx, status, v)
		if err != nil {
			return nil, 0, err
		}
		merged = append(merged, receipts...)
		total += vendorTotal
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].CreatedAt == merged[j].CreatedAt {
			return merged[i].ID > merged[j].ID
		}
		return merged[i].CreatedAt > merged[j].CreatedAt
	})
	if matchedOnly != nil {
		merged = filterReceiptsByMatch(merged, *matchedOnly)
		total = len(merged)
	}
	if offset >= len(merged) {
		return []*Receipt{}, total, nil
	}
	end := offset + limit
	if end > len(merged) {
		end = len(merged)
	}
	return merged[offset:end], total, nil
}

func (s *Service) listAllReceiptsPages(ctx context.Context, status, vendor string) ([]*Receipt, int, error) {
	const pageSize = 100
	merged := []*Receipt{}
	total := 0
	for offset := 0; ; {
		page, vendorTotal, err := s.listReceiptsPage(ctx, status, vendor, pageSize, offset)
		if err != nil {
			return nil, 0, err
		}
		total = vendorTotal
		merged = append(merged, page...)
		if len(page) == 0 || offset+len(page) >= vendorTotal {
			break
		}
		offset += len(page)
	}
	return merged, total, nil
}

func filterReceiptsByMatch(receipts []*Receipt, matched bool) []*Receipt {
	out := make([]*Receipt, 0, len(receipts))
	for _, r := range receipts {
		if r.IsMatched() == matched {
			out = append(out, r)
		}
	}
	return out
}

func (s *Service) GetReceipt(ctx context.Context, id string) (*Receipt, error) {
	var resp struct {
		RetailSync *rawReceipt `json:"retailSync"`
	}
	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_RetailSyncQuery",
		Query:         GetReceiptQuery,
		Variables:     map[string]any{"syncId": id},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.RetailSync == nil {
		return nil, errors.New(errors.ResourceNotFound, "receipt not found", errors.CatAPI, false, nil)
	}
	return toReceipt(resp.RetailSync), nil
}

func (s *Service) DeleteReceipt(ctx context.Context, id string) error {
	var resp struct {
		DeleteUnmatchedRetailSync struct {
			Success bool                 `json:"success"`
			Errors  receiptPayloadErrors `json:"errors"`
		} `json:"deleteUnmatchedRetailSync"`
	}
	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_DeleteRetailSync",
		Query:         DeleteReceiptMutation,
		Variables:     map[string]any{"syncId": id},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := resp.DeleteUnmatchedRetailSync.Errors.toError("failed to delete receipt"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteUnmatchedRetailSync.Success {
		return errors.New(errors.APIError, "failed to delete receipt", errors.CatAPI, false, nil)
	}
	return nil
}

func (s *Service) matchableRetailTransactionID(ctx context.Context, receiptID string, wantLinked bool) (string, error) {
	receipt, err := s.GetReceipt(ctx, receiptID)
	if err != nil {
		return "", err
	}
	seen := false
	for i := range receipt.Orders {
		for j := range receipt.Orders[i].Transactions {
			seen = true
			linked := receipt.Orders[i].Transactions[j].Linked != nil
			if linked == wantLinked {
				return receipt.Orders[i].Transactions[j].ID, nil
			}
		}
	}
	if !seen {
		return "", errors.New(errors.APIError, "receipt does not include a transaction to match", errors.CatAPI, false, nil)
	}
	if wantLinked {
		return "", errors.New(errors.APIError, "receipt is not matched to any transaction", errors.CatAPI, false, nil)
	}
	return "", errors.New(errors.APIError, "receipt is already matched", errors.CatAPI, false, nil)
}

func (s *Service) MatchReceipt(ctx context.Context, receiptID, transactionID string) (*Receipt, error) {
	retailTransactionID, err := s.matchableRetailTransactionID(ctx, receiptID, false)
	if err != nil {
		return nil, err
	}
	var resp struct {
		MatchRetailTransaction struct {
			Errors receiptPayloadErrors `json:"errors"`
		} `json:"matchRetailTransaction"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_MatchRetailTransaction",
		Query:         MatchReceiptMutation,
		Variables: map[string]any{
			"retailTransactionId": retailTransactionID,
			"transactionId":       transactionID,
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.MatchRetailTransaction.Errors.toError("failed to match receipt"); apiErr != nil {
		return nil, apiErr
	}
	return s.GetReceipt(ctx, receiptID)
}

func (s *Service) UnmatchReceipt(ctx context.Context, receiptID string) (*Receipt, error) {
	retailTransactionID, err := s.matchableRetailTransactionID(ctx, receiptID, true)
	if err != nil {
		return nil, err
	}
	var resp struct {
		UnmatchRetailTransaction struct {
			Errors receiptPayloadErrors `json:"errors"`
		} `json:"unmatchRetailTransaction"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UnmatchRetailTransaction",
		Query:         UnmatchReceiptMutation,
		Variables:     map[string]any{"retailTransactionId": retailTransactionID},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.UnmatchRetailTransaction.Errors.toError("failed to unmatch receipt"); apiErr != nil {
		return nil, apiErr
	}
	return s.GetReceipt(ctx, receiptID)
}

func (s *Service) UpdateReceipt(ctx context.Context, receiptID string, update *ReceiptUpdate) (*Receipt, error) {
	receipt, err := s.GetReceipt(ctx, receiptID)
	if err != nil {
		return nil, err
	}
	if len(receipt.Orders) == 0 {
		return nil, errors.New(errors.APIError, "receipt does not include an order to update", errors.CatAPI, false, nil)
	}
	input := map[string]any{"retailOrderId": receipt.Orders[0].ID}
	if update.MerchantName != nil {
		input["merchantName"] = *update.MerchantName
	}
	if update.Date != nil {
		input["date"] = *update.Date
	}
	if update.TotalBeforeTax != nil {
		input["totalBeforeTax"] = *update.TotalBeforeTax
	}
	if update.Tax != nil {
		input["tax"] = *update.Tax
	}
	if update.Tip != nil {
		input["tip"] = *update.Tip
	}
	if update.GrandTotal != nil {
		input["grandTotal"] = *update.GrandTotal
	}
	var resp struct {
		UpdateRetailOrder struct {
			RetailSync *rawReceipt          `json:"retailSync"`
			Errors     receiptPayloadErrors `json:"errors"`
		} `json:"updateRetailOrder"`
	}
	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateRetailOrder",
		Query:         UpdateReceiptMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.UpdateRetailOrder.Errors.toError("failed to update receipt"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateRetailOrder.RetailSync == nil {
		return nil, errors.New(errors.APISchemaChanged, "receipt update response missing retailSync", errors.CatAPI, false, nil)
	}
	return toReceipt(resp.UpdateRetailOrder.RetailSync), nil
}

func (s *Service) GetReceiptSettings(ctx context.Context) ([]ReceiptSettings, error) {
	var resp struct {
		RetailExtensionSettings *struct {
			RetailVendorSettings []struct {
				Vendor                               string `json:"vendor"`
				ShouldCategorizeAndSplitTransactions *bool  `json:"shouldCategorizeAndSplitTransactions"`
				ShouldUpdateTransactionsNotes        *bool  `json:"shouldUpdateTransactionsNotes"`
			} `json:"retailVendorSettings"`
		} `json:"retailExtensionSettings"`
	}
	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetRetailExtensionSettings",
		Query:         GetReceiptSettingsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.RetailExtensionSettings == nil {
		return []ReceiptSettings{}, nil
	}
	out := make([]ReceiptSettings, 0, len(resp.RetailExtensionSettings.RetailVendorSettings))
	for _, vs := range resp.RetailExtensionSettings.RetailVendorSettings {
		out = append(out, ReceiptSettings{
			Vendor:                 vs.Vendor,
			AutoCategorizeAndSplit: vs.ShouldCategorizeAndSplitTransactions,
			UpdateTransactionNotes: vs.ShouldUpdateTransactionsNotes,
		})
	}
	return out, nil
}

func (s *Service) UpdateReceiptSettings(ctx context.Context, autoCategorize, updateNotes *bool) (*ReceiptSettings, error) {
	input := map[string]any{"vendor": "user_import"}
	if autoCategorize != nil {
		input["shouldCategorizeAndSplitTransactions"] = *autoCategorize
	}
	if updateNotes != nil {
		input["shouldUpdateTransactionsNotes"] = *updateNotes
	}
	var resp struct {
		UpdateRetailVendorSettings struct {
			RetailVendorSettings *struct {
				Vendor                               string `json:"vendor"`
				ShouldCategorizeAndSplitTransactions *bool  `json:"shouldCategorizeAndSplitTransactions"`
				ShouldUpdateTransactionsNotes        *bool  `json:"shouldUpdateTransactionsNotes"`
			} `json:"retailVendorSettings"`
			Errors receiptPayloadErrors `json:"errors"`
		} `json:"updateRetailVendorSettings"`
	}
	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Common_UpdateRetailVendorSettings",
		Query:         UpdateReceiptSettingsMutation,
		Variables:     map[string]any{"input": input},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := resp.UpdateRetailVendorSettings.Errors.toError("failed to update receipt settings"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateRetailVendorSettings.RetailVendorSettings == nil {
		return nil, errors.New(errors.APISchemaChanged, "receipt settings update response missing retailVendorSettings", errors.CatAPI, false, nil)
	}
	vs := resp.UpdateRetailVendorSettings.RetailVendorSettings
	return &ReceiptSettings{
		Vendor:                 vs.Vendor,
		AutoCategorizeAndSplit: vs.ShouldCategorizeAndSplitTransactions,
		UpdateTransactionNotes: vs.ShouldUpdateTransactionsNotes,
	}, nil
}

func (s *Service) DownloadReceipt(ctx context.Context, id string, w io.Writer) (string, error) {
	receipt, err := s.GetReceipt(ctx, id)
	if err != nil {
		return "", err
	}
	if len(receipt.Attachments) == 0 || receipt.Attachments[0].OriginalAssetURL == "" {
		return "", errors.New(errors.ResourceNotFound, "receipt has no downloadable image", errors.CatAPI, false, nil)
	}
	attachment := receipt.Attachments[0]
	if err := s.DownloadAttachment(ctx, attachment.OriginalAssetURL, w); err != nil {
		return "", err
	}
	filename := filepath.Base(attachment.Filename)
	if filename == "" || filename == "." {
		filename = id
		if attachment.Extension != "" {
			filename += "." + attachment.Extension
		}
	}
	return filename, nil
}
