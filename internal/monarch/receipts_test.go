package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

const testRawReceiptLinked = `{"id":"sync-1","vendor":"user_import","status":"pending_matches","startedAt":"2026-05-01T00:00:00Z","endedAt":null,"createdAt":"2026-05-01T00:00:00Z","updatedAt":"2026-05-01T00:00:00Z","orders":[{"id":"order-1","merchantName":"Whole Foods","date":"2026-05-01","totalForProducts":42.5,"tax":3.5,"tip":1.0,"grandTotal":46.0,"displayStatus":"pending","retailLineItems":[{"id":"li-1","title":"Milk","quantity":2,"price":5.0,"total":10.0,"isAssociatedToRetailTransaction":true,"category":{"id":"cat-1","name":"Groceries"}},{"id":"li-2","title":"Mystery"}],"retailTransactions":[{"id":"rt-1","date":"2026-05-01","total":46.0,"transactionType":"payment","transactionUpdateSkipped":true,"transaction":{"id":"tx-9","isManual":true,"hasSplitTransactions":true,"merchant":{"name":"Whole Foods"}}},{"id":"rt-2","date":"2026-05-01","total":46.0,"transactionType":"payment","transaction":null}]}],"attachments":[{"id":"att-1","storageId":"s-1","filename":"receipt.jpg","extension":"jpg","sizeBytes":1024,"originalAssetUrl":"https://example.com/receipt.jpg","thumbnailUrl":"https://example.com/thumb.jpg"}]}`

const testRawReceiptBare = `{"id":"sync-2","vendor":"email_import","status":"pending","startedAt":null,"endedAt":null,"createdAt":"2026-04-01T00:00:00Z","updatedAt":"2026-04-01T00:00:00Z","orders":[],"attachments":[]}`

func receiptMockService(t *testing.T, handler func(req *graphql.Request, result any) error) *Service {
	t.Helper()
	return NewService(&mockClient{token: "token-123", handler: handler})
}

func mustReceiptPayload(t *testing.T, result any, payload string) {
	t.Helper()
	if err := json.Unmarshal([]byte(payload), result); err != nil {
		t.Fatalf("fixture unmarshal: %v", err)
	}
}

func TestReceiptPayloadErrorsDecoding(t *testing.T) {
	decode := func(raw string) (receiptPayloadErrors, error) {
		var e receiptPayloadErrors
		err := json.Unmarshal([]byte(raw), &e)
		return e, err
	}

	if _, err := decode(`null`); err != nil {
		t.Fatalf("null: %v", err)
	}
	e, err := decode(`{}`)
	mustNoErr(t, err)
	if got := e.toError("fallback"); got != nil {
		t.Fatalf("empty object should decode to no error, got %v", got)
	}
	e, err = decode(`{"message":"boom","code":"E1"}`)
	mustNoErr(t, err)
	if got := e.toError("fallback"); got == nil || !strings.Contains(got.Error(), "boom") {
		t.Fatalf("message error = %v", got)
	}
	e, err = decode(`{"fieldErrors":[{"field":"name","messages":["too short","banned"]}]}`)
	mustNoErr(t, err)
	if got := e.toError("fallback"); got == nil || !strings.Contains(got.Error(), "name: too short") {
		t.Fatalf("fieldErrors error = %v", got)
	}
	e, err = decode(`[{"message":"first"},{"message":"second"}]`)
	mustNoErr(t, err)
	if len(e.Items) != 2 {
		t.Fatalf("array items = %d", len(e.Items))
	}
	e, err = decode(`{"code":"ONLY_CODE"}`)
	mustNoErr(t, err)
	if got := e.toError("fallback"); got == nil || !strings.Contains(got.Error(), "fallback") {
		t.Fatalf("code-only error = %v", got)
	}
	if _, err := decode(`{invalid`); err == nil {
		t.Fatal("expected decode error for invalid JSON")
	}
	if _, err := decode(`[invalid`); err == nil {
		t.Fatal("expected decode error for invalid array JSON")
	}
}

func TestToReceiptEdges(t *testing.T) {
	var raw rawReceipt
	mustReceiptPayload(t, &raw, testRawReceiptLinked)
	r := toReceipt(&raw)
	if r.ID != "sync-1" || r.Vendor != "user_import" {
		t.Fatalf("receipt = %+v", r)
	}
	if len(r.Orders) != 1 || len(r.Orders[0].LineItems) != 2 {
		t.Fatalf("orders = %+v", r.Orders)
	}
	if r.Orders[0].LineItems[1].CategoryID != "" {
		t.Fatalf("bare line item should have no category: %+v", r.Orders[0].LineItems[1])
	}
	if r.Orders[0].Transactions[1].Linked != nil {
		t.Fatalf("second transaction should be unlinked")
	}
	if !r.IsMatched() {
		t.Fatal("receipt with a linked transaction should be matched")
	}
	if ids := r.MatchedTransactionIDs(); len(ids) != 1 || ids[0] != "tx-9" {
		t.Fatalf("matched ids = %v", ids)
	}

	var bare rawReceipt
	mustReceiptPayload(t, &bare, testRawReceiptBare)
	b := toReceipt(&bare)
	if b.IsMatched() {
		t.Fatal("bare receipt should be unmatched")
	}
	if ids := b.MatchedTransactionIDs(); len(ids) != 0 {
		t.Fatalf("matched ids = %v", ids)
	}
}

func TestListReceipts(t *testing.T) {
	t.Run("single vendor passthrough", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName != "Common_RetailSyncsQueryWithTotal" {
				t.Fatalf("operation = %q", req.OperationName)
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptLinked+`]}}`), result)
		})
		list, total, err := svc.ListReceipts(context.Background(), "pending", "user_import", 10, 0, nil)
		mustNoErr(t, err)
		mustLen(t, list, 1)
		eq(t, 1, total)
		eq(t, "sync-1", list[0].ID)
	})

	t.Run("merged vendors sorted newest first", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			filters, _ := req.Variables["filters"].(map[string]any)
			if filters["vendor"] == "user_import" {
				return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptLinked+`]}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptBare+`]}}`), result)
		})
		list, total, err := svc.ListReceipts(context.Background(), "", "", 10, 0, nil)
		mustNoErr(t, err)
		eq(t, 2, total)
		mustLen(t, list, 2)
		eq(t, "sync-1", list[0].ID)
		eq(t, "sync-2", list[1].ID)
	})

	t.Run("offset beyond merged returns empty", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			filters, _ := req.Variables["filters"].(map[string]any)
			if filters["vendor"] == "user_import" {
				return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptLinked+`]}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":0,"results":[]}}`), result)
		})
		list, total, err := svc.ListReceipts(context.Background(), "", "", 10, 5, nil)
		mustNoErr(t, err)
		eq(t, 1, total)
		mustLen(t, list, 0)
	})

	t.Run("matched filter scans all pages", func(t *testing.T) {
		var calls int
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			calls++
			filters, _ := req.Variables["filters"].(map[string]any)
			if filters["vendor"] == "user_import" {
				return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptLinked+`]}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":1,"results":[`+testRawReceiptBare+`]}}`), result)
		})
		matched := true
		list, total, err := svc.ListReceipts(context.Background(), "", "", 10, 0, &matched)
		mustNoErr(t, err)
		eq(t, 1, total)
		mustLen(t, list, 1)
		eq(t, "sync-1", list[0].ID)

		unmatched := false
		list, total, err = svc.ListReceipts(context.Background(), "", "", 10, 0, &unmatched)
		mustNoErr(t, err)
		eq(t, 1, total)
		mustLen(t, list, 1)
		eq(t, "sync-2", list[0].ID)
		if calls != 4 {
			t.Fatalf("expected 4 page fetches, got %d", calls)
		}
	})

	t.Run("client error propagates", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, _, err := svc.ListReceipts(context.Background(), "", "user_import", 10, 0, nil)
		hasErr(t, err)
	})

	t.Run("defaults normalize limit and offset", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.Variables["limit"] != 100 {
				t.Fatalf("limit = %v", req.Variables["limit"])
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":0,"results":[]}}`), result)
		})
		_, _, err := svc.ListReceipts(context.Background(), "", "user_import", 0, -5, nil)
		mustNoErr(t, err)
	})
}

func TestGetReceipt(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		got, err := svc.GetReceipt(context.Background(), "sync-1")
		mustNoErr(t, err)
		eq(t, "sync-1", got.ID)
	})
	t.Run("missing", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":null}`), result)
		})
		_, err := svc.GetReceipt(context.Background(), "nope")
		hasErr(t, err)
	})
}

func TestDeleteReceipt(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName != "Common_DeleteRetailSync" {
				t.Fatalf("operation = %q", req.OperationName)
			}
			return json.Unmarshal([]byte(`{"deleteUnmatchedRetailSync":{"success":true,"errors":null}}`), result)
		})
		mustNoErr(t, svc.DeleteReceipt(context.Background(), "sync-1"))
	})
	t.Run("payload message error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"deleteUnmatchedRetailSync":{"success":false,"errors":{"message":"cannot delete matched"}}}`), result)
		})
		err := svc.DeleteReceipt(context.Background(), "sync-1")
		hasErr(t, err)
		if !strings.Contains(err.Error(), "cannot delete matched") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("payload array error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"deleteUnmatchedRetailSync":{"success":false,"errors":[{"message":"gone"}]}}`), result)
		})
		hasErr(t, svc.DeleteReceipt(context.Background(), "sync-1"))
	})
	t.Run("success false without errors", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"deleteUnmatchedRetailSync":{"success":false,"errors":null}}`), result)
		})
		hasErr(t, svc.DeleteReceipt(context.Background(), "sync-1"))
	})
}

func TestMatchReceipt(t *testing.T) {
	t.Run("picks first unmatched transaction", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "Common_RetailSyncQuery":
				return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
			case "Common_MatchRetailTransaction":
				if req.Variables["retailTransactionId"] != "rt-2" {
					t.Fatalf("retailTransactionId = %v, want rt-2", req.Variables["retailTransactionId"])
				}
				return json.Unmarshal([]byte(`{"matchRetailTransaction":{"retailSync":{"id":"sync-1"},"errors":null}}`), result)
			default:
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			}
		})
		got, err := svc.MatchReceipt(context.Background(), "sync-1", "tx-9")
		mustNoErr(t, err)
		eq(t, "sync-1", got.ID)
	})
	t.Run("already matched", func(t *testing.T) {
		linked := strings.Replace(testRawReceiptLinked, `"transaction":null`, `"transaction":{"id":"tx-1"}`, 1)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+linked+`}`), result)
		})
		_, err := svc.MatchReceipt(context.Background(), "sync-1", "tx-9")
		hasErr(t, err)
		if !strings.Contains(err.Error(), "already matched") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("no transactions", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptBare+`}`), result)
		})
		_, err := svc.MatchReceipt(context.Background(), "sync-2", "tx-9")
		hasErr(t, err)
	})
	t.Run("mutation error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_MatchRetailTransaction" {
				return json.Unmarshal([]byte(`{"matchRetailTransaction":{"errors":{"message":"nope"}}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		// rt-2 is the unmatched one; make fixture with only unlinked
		_ = svc
		svc2 := receiptMockService(t, func(req *graphql.Request, result any) error {
			bare := strings.Replace(testRawReceiptLinked, `"transaction":{"id":"tx-9","isManual":true,"hasSplitTransactions":true,"merchant":{"name":"Whole Foods"}}`, `"transaction":null`, 1)
			if req.OperationName == "Common_RetailSyncQuery" {
				return json.Unmarshal([]byte(`{"retailSync":`+bare+`}`), result)
			}
			return json.Unmarshal([]byte(`{"matchRetailTransaction":{"errors":{"message":"nope"}}}`), result)
		})
		_, err := svc2.MatchReceipt(context.Background(), "sync-1", "tx-9")
		hasErr(t, err)
	})
}

func TestUnmatchReceipt(t *testing.T) {
	t.Run("picks linked transaction", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "Common_RetailSyncQuery":
				return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
			case "Web_UnmatchRetailTransaction":
				if req.Variables["retailTransactionId"] != "rt-1" {
					t.Fatalf("retailTransactionId = %v, want rt-1", req.Variables["retailTransactionId"])
				}
				return json.Unmarshal([]byte(`{"unmatchRetailTransaction":{"retailSync":{"id":"sync-1","status":"pending"},"errors":null}}`), result)
			default:
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			}
		})
		got, err := svc.UnmatchReceipt(context.Background(), "sync-1")
		mustNoErr(t, err)
		eq(t, "sync-1", got.ID)
	})
	t.Run("nothing linked", func(t *testing.T) {
		bare := strings.Replace(testRawReceiptLinked, `"transaction":{"id":"tx-9","isManual":true,"hasSplitTransactions":true,"merchant":{"name":"Whole Foods"}}`, `"transaction":null`, 1)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+bare+`}`), result)
		})
		_, err := svc.UnmatchReceipt(context.Background(), "sync-1")
		hasErr(t, err)
		if !strings.Contains(err.Error(), "not matched") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("mutation error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Web_UnmatchRetailTransaction" {
				return json.Unmarshal([]byte(`{"unmatchRetailTransaction":{"errors":[{"message":"locked"}]}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		_, err := svc.UnmatchReceipt(context.Background(), "sync-1")
		hasErr(t, err)
	})
}

func TestUpdateReceipt(t *testing.T) {
	merchant, total := "New Mart", 50.0
	full := &ReceiptUpdate{MerchantName: &merchant, GrandTotal: &total}
	t.Run("all fields sent", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "Common_RetailSyncQuery":
				return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
			case "Common_UpdateRetailOrder":
				input, _ := req.Variables["input"].(map[string]any)
				if input["retailOrderId"] != "order-1" || input["merchantName"] != "New Mart" || input["grandTotal"] != 50.0 {
					t.Fatalf("input = %v", input)
				}
				return json.Unmarshal([]byte(`{"updateRetailOrder":{"retailSync":`+testRawReceiptLinked+`,"errors":null}}`), result)
			default:
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			}
		})
		got, err := svc.UpdateReceipt(context.Background(), "sync-1", full)
		mustNoErr(t, err)
		eq(t, "sync-1", got.ID)
	})
	t.Run("no orders", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptBare+`}`), result)
		})
		_, err := svc.UpdateReceipt(context.Background(), "sync-2", &ReceiptUpdate{})
		hasErr(t, err)
	})
	t.Run("mutation error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_UpdateRetailOrder" {
				return json.Unmarshal([]byte(`{"updateRetailOrder":{"errors":{"message":"bad totals"}}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		_, err := svc.UpdateReceipt(context.Background(), "sync-1", &ReceiptUpdate{})
		hasErr(t, err)
	})
	t.Run("missing retailSync in response", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_UpdateRetailOrder" {
				return json.Unmarshal([]byte(`{"updateRetailOrder":{"retailSync":null,"errors":null}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		_, err := svc.UpdateReceipt(context.Background(), "sync-1", &ReceiptUpdate{})
		hasErr(t, err)
	})
}

func TestReceiptSettingsService(t *testing.T) {
	t.Run("list settings", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName != "Common_GetRetailExtensionSettings" {
				t.Fatalf("operation = %q", req.OperationName)
			}
			return json.Unmarshal([]byte(`{"retailExtensionSettings":{"id":"ext-1","retailVendorSettings":[{"id":"vs-1","vendor":"user_import","shouldCategorizeAndSplitTransactions":true,"shouldUpdateTransactionsNotes":false}]}}`), result)
		})
		got, err := svc.GetReceiptSettings(context.Background())
		mustNoErr(t, err)
		mustLen(t, got, 1)
		eq(t, "user_import", got[0].Vendor)
	})
	t.Run("nil extension settings", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailExtensionSettings":null}`), result)
		})
		got, err := svc.GetReceiptSettings(context.Background())
		mustNoErr(t, err)
		mustLen(t, got, 0)
	})
	t.Run("update settings", func(t *testing.T) {
		auto, notes := true, false
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName != "Common_UpdateRetailVendorSettings" {
				t.Fatalf("operation = %q", req.OperationName)
			}
			input, _ := req.Variables["input"].(map[string]any)
			if input["vendor"] != "user_import" || input["shouldCategorizeAndSplitTransactions"] != true {
				t.Fatalf("input = %v", input)
			}
			return json.Unmarshal([]byte(`{"updateRetailVendorSettings":{"retailVendorSettings":{"id":"vs-1","vendor":"user_import","shouldCategorizeAndSplitTransactions":true,"shouldUpdateTransactionsNotes":false},"errors":null}}`), result)
		})
		got, err := svc.UpdateReceiptSettings(context.Background(), &auto, &notes)
		mustNoErr(t, err)
		eq(t, "user_import", got.Vendor)
	})
	t.Run("update settings error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"updateRetailVendorSettings":{"errors":{"message":"denied"}}}`), result)
		})
		_, err := svc.UpdateReceiptSettings(context.Background(), nil, nil)
		hasErr(t, err)
	})
	t.Run("update settings missing payload", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"updateRetailVendorSettings":{"retailVendorSettings":null,"errors":null}}`), result)
		})
		_, err := svc.UpdateReceiptSettings(context.Background(), nil, nil)
		hasErr(t, err)
	})
}

func TestDownloadReceipt(t *testing.T) {
	swapImageTransport := func(t *testing.T, status int, body string) {
		t.Helper()
		orig := http.DefaultTransport
		t.Cleanup(func() { http.DefaultTransport = orig })
		http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "example.com" {
				t.Fatalf("unexpected download host %q", req.URL.Host)
			}
			resp := testutil.JSONResponse(body)
			resp.StatusCode = status
			return resp, nil
		})
	}

	t.Run("happy path", func(t *testing.T) {
		swapImageTransport(t, 200, "fake-image-bytes")
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName != "Common_RetailSyncQuery" {
				t.Fatalf("operation = %q", req.OperationName)
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		var buf bytes.Buffer
		filename, err := svc.DownloadReceipt(context.Background(), "sync-1", &buf)
		mustNoErr(t, err)
		eq(t, "receipt.jpg", filename)
		eq(t, "fake-image-bytes", buf.String())
	})

	t.Run("sanitizes traversal filename", func(t *testing.T) {
		swapImageTransport(t, 200, "x")
		evil := strings.Replace(testRawReceiptLinked, `"filename":"receipt.jpg"`, `"filename":"../../evil.jpg"`, 1)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+evil+`}`), result)
		})
		var buf bytes.Buffer
		filename, err := svc.DownloadReceipt(context.Background(), "sync-1", &buf)
		mustNoErr(t, err)
		eq(t, "evil.jpg", filename)
	})

	t.Run("falls back to id plus extension", func(t *testing.T) {
		swapImageTransport(t, 200, "x")
		noname := strings.Replace(testRawReceiptLinked, `"filename":"receipt.jpg"`, `"filename":""`, 1)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+noname+`}`), result)
		})
		var buf bytes.Buffer
		filename, err := svc.DownloadReceipt(context.Background(), "sync-1", &buf)
		mustNoErr(t, err)
		eq(t, "sync-1.jpg", filename)
	})

	t.Run("no attachments", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptBare+`}`), result)
		})
		var buf bytes.Buffer
		_, err := svc.DownloadReceipt(context.Background(), "sync-2", &buf)
		hasErr(t, err)
	})

	t.Run("download error", func(t *testing.T) {
		swapImageTransport(t, 500, "boom")
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		var buf bytes.Buffer
		_, err := svc.DownloadReceipt(context.Background(), "sync-1", &buf)
		hasErr(t, err)
	})

	t.Run("lookup error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"retailSync":null}`), result)
		})
		var buf bytes.Buffer
		_, err := svc.DownloadReceipt(context.Background(), "nope", &buf)
		hasErr(t, err)
	})
}

func TestUploadReceiptToInbox(t *testing.T) {
	writeTempReceipt := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "receipt.jpg")
		if err := os.WriteFile(path, []byte("jpg-bytes"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		return path
	}
	swapUploadTransport := func(t *testing.T, status int, check func(body string)) {
		t.Helper()
		orig := retailSyncUploadClient
		t.Cleanup(func() { retailSyncUploadClient = orig })
		retailSyncUploadClient = &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/retail-sync/sync-1/files") {
				t.Fatalf("upload path = %q", req.URL.Path)
			}
			if check != nil {
				var buf bytes.Buffer
				if _, err := buf.ReadFrom(req.Body); err != nil {
					t.Fatalf("read body: %v", err)
				}
				check(buf.String())
			}
			resp := testutil.JSONResponse(`{}`)
			resp.StatusCode = status
			return resp, nil
		})}
	}

	t.Run("happy path", func(t *testing.T) {
		swapUploadTransport(t, 200, func(body string) {
			if !strings.Contains(body, `filename="receipt.jpg"`) || !strings.Contains(body, `"vendor":"user_import"`) {
				t.Fatalf("upload body missing fields: %s", body)
			}
		})
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "Common_CreateBulkRetailSync":
				return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1","vendor":"user_import","status":"created"}],"errors":null}}`), result)
			case "Common_StartRetailSync":
				if req.Variables["syncId"] != "sync-1" {
					t.Fatalf("syncId = %v", req.Variables["syncId"])
				}
				return json.Unmarshal([]byte(`{"startRetailSync":{"retailSync":{"id":"sync-1","vendor":"user_import","status":"started","startedAt":"2026-05-01T00:00:00Z"},"errors":null}}`), result)
			default:
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			}
		})
		got, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		mustNoErr(t, err)
		eq(t, "sync-1", got.ID)
		eq(t, "started", got.Status)
	})

	t.Run("read error", func(t *testing.T) {
		svc := receiptMockService(t, nil)
		_, err := svc.UploadReceiptToInbox(context.Background(), filepath.Join(t.TempDir(), "missing.jpg"))
		hasErr(t, err)
	})

	t.Run("create error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
	})

	t.Run("empty syncs", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[],"errors":null}}`), result)
		})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
	})

	t.Run("create payload error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[],"errors":{"message":"quota"}}}`), result)
		})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
		if !strings.Contains(err.Error(), "quota") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("upload non-200", func(t *testing.T) {
		swapUploadTransport(t, 500, nil)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`), result)
		})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
	})

	t.Run("upload transport error", func(t *testing.T) {
		orig := retailSyncUploadClient
		t.Cleanup(func() { retailSyncUploadClient = orig })
		retailSyncUploadClient = &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("dial failed")
		})}
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`), result)
		})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
	})

	t.Run("start error", func(t *testing.T) {
		swapUploadTransport(t, 200, nil)
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_StartRetailSync" {
				return json.Unmarshal([]byte(`{"startRetailSync":{"retailSync":null,"errors":{"message":"stuck"}}}`), result)
			}
			return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`), result)
		})
		_, err := svc.UploadReceiptToInbox(context.Background(), writeTempReceipt(t))
		hasErr(t, err)
	})
}

func TestReceiptCoverageGaps(t *testing.T) {
	t.Run("unknown extension defaults content type", func(t *testing.T) {
		var seenContentType string
		orig := retailSyncUploadClient
		t.Cleanup(func() { retailSyncUploadClient = orig })
		retailSyncUploadClient = &http.Client{Transport: testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			seenContentType = req.Header.Get("Content-Type")
			resp := testutil.JSONResponse(`{}`)
			resp.StatusCode = 200
			return resp, nil
		})}
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "Common_CreateBulkRetailSync":
				return json.Unmarshal([]byte(`{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`), result)
			case "Common_StartRetailSync":
				return json.Unmarshal([]byte(`{"startRetailSync":{"retailSync":{"id":"sync-1","status":"started"},"errors":null}}`), result)
			default:
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			}
		})
		path := filepath.Join(t.TempDir(), "receipt-noext")
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		_, err := svc.UploadReceiptToInbox(context.Background(), path)
		mustNoErr(t, err)
		if !strings.Contains(seenContentType, "multipart/form-data") {
			t.Fatalf("content-type = %q", seenContentType)
		}
	})

	t.Run("multi page loop", func(t *testing.T) {
		makePage := func(start, n int) string {
			items := make([]string, 0, n)
			for i := 0; i < n; i++ {
				items = append(items, `{"id":"s-`+strconv.Itoa(start+i)+`","vendor":"user_import","status":"pending","createdAt":"2026-05-01T00:00:00Z"}`)
			}
			return strings.Join(items, ",")
		}
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			offset, _ := req.Variables["offset"].(int)
			if offset == 0 {
				return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":150,"results":[`+makePage(0, 100)+`]}}`), result)
			}
			return json.Unmarshal([]byte(`{"retailSyncsWithTotal":{"totalCount":150,"results":[`+makePage(100, 50)+`]}}`), result)
		})
		matched := false
		list, total, err := svc.ListReceipts(context.Background(), "", "user_import", 200, 0, &matched)
		mustNoErr(t, err)
		eq(t, 150, total)
		mustLen(t, list, 150)
	})

	t.Run("page loop client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		matched := true
		_, _, err := svc.ListReceipts(context.Background(), "", "", 10, 0, &matched)
		hasErr(t, err)
	})

	t.Run("get client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, err := svc.GetReceipt(context.Background(), "x")
		hasErr(t, err)
	})

	t.Run("delete client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		hasErr(t, svc.DeleteReceipt(context.Background(), "x"))
	})

	t.Run("match lookup error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, err := svc.MatchReceipt(context.Background(), "x", "tx-1")
		hasErr(t, err)
	})

	t.Run("match mutation client error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_MatchRetailTransaction" {
				return fmt.Errorf("boom")
			}
			bare := strings.Replace(testRawReceiptLinked, `"transaction":{"id":"tx-9","isManual":true,"hasSplitTransactions":true,"merchant":{"name":"Whole Foods"}}`, `"transaction":null`, 1)
			return json.Unmarshal([]byte(`{"retailSync":`+bare+`}`), result)
		})
		_, err := svc.MatchReceipt(context.Background(), "sync-1", "tx-1")
		hasErr(t, err)
	})

	t.Run("unmatch mutation client error", func(t *testing.T) {
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Web_UnmatchRetailTransaction" {
				return fmt.Errorf("boom")
			}
			return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
		})
		_, err := svc.UnmatchReceipt(context.Background(), "sync-1")
		hasErr(t, err)
	})

	t.Run("update all scalar fields", func(t *testing.T) {
		merchant, date := "M", "2026-05-02"
		sub, tax, tip, total := 1.0, 2.0, 3.0, 40.0
		svc := receiptMockService(t, func(req *graphql.Request, result any) error {
			if req.OperationName == "Common_RetailSyncQuery" {
				return json.Unmarshal([]byte(`{"retailSync":`+testRawReceiptLinked+`}`), result)
			}
			input, _ := req.Variables["input"].(map[string]any)
			for _, k := range []string{"merchantName", "date", "totalBeforeTax", "tax", "tip", "grandTotal"} {
				if _, ok := input[k]; !ok {
					t.Fatalf("input missing %q: %v", k, input)
				}
			}
			return json.Unmarshal([]byte(`{"updateRetailOrder":{"retailSync":`+testRawReceiptLinked+`,"errors":null}}`), result)
		})
		_, err := svc.UpdateReceipt(context.Background(), "sync-1", &ReceiptUpdate{
			MerchantName: &merchant, Date: &date, TotalBeforeTax: &sub,
			Tax: &tax, Tip: &tip, GrandTotal: &total,
		})
		mustNoErr(t, err)
	})

	t.Run("update lookup error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, err := svc.UpdateReceipt(context.Background(), "x", &ReceiptUpdate{})
		hasErr(t, err)
	})

	t.Run("settings client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: func(req *graphql.Request, result any) error {
			return fmt.Errorf("boom")
		}})
		_, err := svc.GetReceiptSettings(context.Background())
		hasErr(t, err)
		_, err = svc.UpdateReceiptSettings(context.Background(), nil, nil)
		hasErr(t, err)
	})
}
