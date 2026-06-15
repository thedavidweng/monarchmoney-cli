package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

func withCacheCommandTestDefaults(t *testing.T, sessionPath, cachePath string) *int {
	t.Helper()

	oldExitFunc := exitFunc
	oldDefaultSessionPath := defaultSessionPath
	oldJSONMode := jsonMode
	oldPretty := pretty
	oldProfile := profile
	oldTransport := http.DefaultTransport
	oldSyncFrom := syncFrom
	oldSyncLimit := syncLimit
	oldSyncAll := syncAll
	oldCleanupBefore := cleanupBefore

	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defaultSessionPath = func() string { return sessionPath }
	jsonMode = true
	pretty = false
	profile = "default"
	syncFrom = ""
	syncLimit = 1000
	syncAll = false
	cleanupBefore = ""
	cacheSyncCmd.SetContext(context.Background())
	cacheCleanupCmd.SetContext(context.Background())

	viper.Reset()
	viper.Set("cache_path", cachePath)

	t.Cleanup(func() {
		exitFunc = oldExitFunc
		defaultSessionPath = oldDefaultSessionPath
		jsonMode = oldJSONMode
		pretty = oldPretty
		profile = oldProfile
		http.DefaultTransport = oldTransport
		syncFrom = oldSyncFrom
		syncLimit = oldSyncLimit
		syncAll = oldSyncAll
		cleanupBefore = oldCleanupBefore
		viper.Reset()
	})

	return &exitCode
}

func TestCacheSyncPassesFromDateAndPersistsAccountID(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)

	var sawStartDate bool
	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string         `json:"operationName"`
			Variables     map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}

		switch gqlReq.OperationName {
		case "GetAccounts":
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"acc_1","displayName":"Checking","type":{"name":"cash","display":"Cash"},"subtype":{"name":"checking","display":"Checking"},"displayBalance":1250.5,"currentBalance":1250.5,"updatedAt":"2026-05-09T10:00:00Z","displayLastUpdatedAt":"2026-05-09","createdAt":"2026-01-01T00:00:00Z"}]}}`), nil
		case "GetTransactionsList":
			filters, ok := gqlReq.Variables["filters"].(map[string]any)
			if !ok {
				t.Fatalf("filters = %#v, want map", gqlReq.Variables["filters"])
			}
			if filters["startDate"] == "2026-01-01" {
				sawStartDate = true
			}
			return testutil.JSONResponse(`{"data":{"allTransactions":{"results":[{"id":"tx_1","date":"2026-05-09","amount":-12.34,"merchant":{"name":"Cafe"},"category":{"name":"Dining"},"account":{"id":"acc_1"},"notes":"latte"}],"totalCount":1}}}`), nil
		default:
			t.Fatalf("unexpected operation %q", gqlReq.OperationName)
		}
		return nil, nil
	})

	_ = cacheSyncCmd.Flags().Set("from", "2026-01-01")
	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	if !sawStartDate {
		t.Fatal("cache sync did not pass --from as transaction startDate")
	}

	store, err := cache.NewStore(cachePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // test cleanup
	txs, err := store.SearchTransactions("Cafe")
	if err != nil {
		t.Fatalf("SearchTransactions() error = %v", err)
	}
	if len(txs) != 1 || txs[0].AccountID != "acc_1" {
		t.Fatalf("cached transaction = %#v, want account id acc_1", txs)
	}
}

func TestCacheSyncRejectsInvalidFromDate(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("cache sync should validate --from before making API requests")
		return nil, nil
	})

	_ = cacheSyncCmd.Flags().Set("from", "01-01-2026")
	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM-DD") {
		t.Fatalf("output = %q, want date format guidance", out)
	}
}

func TestCacheSyncFailsWhenAccountsAPIFails(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	})

	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, "failed to sync accounts") {
		t.Fatalf("output = %q, want account sync failure", out)
	}
}

func TestCacheSyncFailsWhenTransactionsAPIFails(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "session.json")
	cachePath := filepath.Join(dir, "cache.sqlite")
	exitCode := withCacheCommandTestDefaults(t, sessionPath, cachePath)
	saveTestSession(t, sessionPath)

	http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		var gqlReq struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(req.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		if gqlReq.OperationName == "GetAccounts" {
			return testutil.JSONResponse(`{"data":{"accounts":[{"id":"acc_1","displayName":"Checking","type":{"name":"cash"},"subtype":{"name":"checking"},"displayBalance":1250.5,"updatedAt":"2026-05-09"}]}}`), nil
		}
		if gqlReq.OperationName == "GetTransactionsList" {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
		t.Fatalf("unexpected operation %q", gqlReq.OperationName)
		return nil, nil
	})

	out := captureStdout(t, func() {
		cacheSyncCmd.Run(cacheSyncCmd, nil)
	})

	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want API failure; output=%q", out)
	}
	if !strings.Contains(out, "failed to sync transactions") {
		t.Fatalf("output = %q, want transaction sync failure", out)
	}
}

func TestCacheCleanupUsesConfiguredCachePathAndValidatesDate(t *testing.T) {
	dir := t.TempDir()
	configuredPath := filepath.Join(dir, "configured.sqlite")
	defaultPath := filepath.Join(dir, "default", "monarch.sqlite")
	exitCode := withCacheCommandTestDefaults(t, filepath.Join(dir, "session.json"), configuredPath)

	store, err := cache.NewStore(configuredPath)
	if err != nil {
		t.Fatalf("NewStore(configured) error = %v", err)
	}
	defer store.Close() //nolint:errcheck // test cleanup
	if err := store.SaveTransactions([]cache.Transaction{{
		ID:       "tx_old",
		Date:     time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		Merchant: "Old",
	}}); err != nil {
		t.Fatalf("SaveTransactions() error = %v", err)
	}

	viper.Set("cache_path", configuredPath)
	t.Setenv("HOME", filepath.Join(dir, "home"))
	_ = os.MkdirAll(filepath.Dir(defaultPath), 0700)

	cleanupBefore = "2026-01-01"
	_ = cacheCleanupCmd.Flags().Set("before", cleanupBefore)
	out := captureStdout(t, func() {
		cacheCleanupCmd.Run(cacheCleanupCmd, nil)
	})

	if *exitCode != 0 {
		t.Fatalf("exitCode = %d; output=%q", *exitCode, out)
	}
	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if got := stats["transactions"]; got.(int64) != 0 {
		t.Fatalf("configured cache transactions = %v, want 0", got)
	}

	cleanupBefore = "not-a-date"
	_ = cacheCleanupCmd.Flags().Set("before", cleanupBefore)
	out = captureStdout(t, func() {
		cacheCleanupCmd.Run(cacheCleanupCmd, nil)
	})
	if *exitCode == 0 {
		t.Fatalf("exitCode = 0, want validation failure; output=%q", out)
	}
	if !strings.Contains(out, "YYYY-MM-DD") {
		t.Fatalf("output = %q, want date format guidance", out)
	}
}
