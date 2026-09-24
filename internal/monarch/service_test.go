package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/internal/testutil"
)

type mockClient struct {
	token   string
	mu      sync.Mutex
	lastReq *graphql.Request
	handler func(req *graphql.Request, result any) error
}

type fakeCSVWriter struct {
	failOnCall int
	calls      int
	err        error
}

func (w *fakeCSVWriter) Write(record []string) error {
	w.calls++
	if w.calls == w.failOnCall {
		return errors.New("write failed")
	}
	return nil
}

func (*fakeCSVWriter) Flush() {}

func (w *fakeCSVWriter) Error() error { return w.err }

func (m *mockClient) Do(_ context.Context, req *graphql.Request, result any) error {
	m.mu.Lock()
	m.lastReq = req
	m.mu.Unlock()
	if m.handler != nil {
		return m.handler(req, result)
	}
	return nil
}

func (m *mockClient) DoMutation(_ context.Context, req *graphql.Request, result any) error {
	m.mu.Lock()
	m.lastReq = req
	m.mu.Unlock()
	if m.handler != nil {
		return m.handler(req, result)
	}
	return nil
}

func (m *mockClient) TokenValue() string {
	return m.token
}

func (m *mockClient) respond(result any, payload string) error {
	if result == nil {
		return nil
	}
	return json.Unmarshal([]byte(payload), result)
}

func clientRespond(result any, payload string) error {
	if result == nil {
		return nil
	}
	return json.Unmarshal([]byte(payload), result)
}

func newMockService(token string) *Service {
	return NewService(&mockClient{token: token})
}

func assertReq(t *testing.T, got *graphql.Request, op string) {
	t.Helper()
	if got == nil {
		t.Fatal("request not captured")
	}
	if got.OperationName != op {
		t.Fatalf("OperationName = %q, want %q", got.OperationName, op)
	}
}

func expectVars(t *testing.T, got, want map[string]any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Variables = %#v, want %#v", got, want)
	}
}

func runGraphQLCase(t *testing.T, op string, wantVars map[string]any, payload string, call func(*Service) error) {
	t.Helper()

	var client *mockClient
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result any) error {
			assertReq(t, req, op)
			expectVars(t, req.Variables, wantVars)
			return client.respond(result, payload)
		},
	}

	if err := call(NewService(client)); err != nil {
		t.Fatalf("%s() error = %v", op, err)
	}
}

func runGraphQLErrorCase(t *testing.T, op string, wantVars map[string]any, call func(*Service) error) {
	t.Helper()

	var client *mockClient //nolint:staticcheck // self-referential closure
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result any) error {
			assertReq(t, req, op)
			expectVars(t, req.Variables, wantVars)
			return errors.New("boom")
		},
	}

	if err := call(NewService(client)); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("%s() error = %v, want boom", op, err)
	}
}

func TestServiceTokenValue(t *testing.T) {
	svc := newMockService("abc123")
	if got := svc.Client.TokenValue(); got != "abc123" {
		t.Fatalf("TokenValue() = %q, want %q", got, "abc123")
	}
}

func TestServiceAccountsAndCoreMethods(t *testing.T) {
	testServiceAccountsCoreReadPaths(t)
	testServiceAccountsCoreMutationPaths(t)
	testServiceAccountsCoreHistoryPaths(t)
	testServiceAccountsCoreSnapshotPaths(t)
}

func testServiceAccountsCoreReadPaths(t *testing.T) {
	t.Helper()

	t.Run("list accounts", func(t *testing.T) {
		runGraphQLCase(t, "GetAccounts", nil, `{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank"},"subtype":{"name":"checking"},"displayBalance":42.5,"updatedAt":"2026-05-08"}]}`, func(s *Service) error {
			got, err := s.ListAccounts(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "a1", got[0].ID)
			eq(t, "bank", got[0].AccountType)
			eq(t, 42.5, got[0].DisplayBalance)
			return nil
		})
	})

	t.Run("get account", func(t *testing.T) {
		runGraphQLCase(t, "AccountDetails_getAccount", map[string]any{"id": "acc-1"}, `{"account":{"id":"acc-1","displayName":"Cash","type":{"name":"cash"},"subtype":{"name":"cash"},"displayBalance":9.5,"updatedAt":"2026-05-08"}}`, func(s *Service) error {
			got, err := s.GetAccount(context.Background(), "acc-1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "acc-1", got.ID)
			eq(t, "Cash", got.DisplayName)
			eq(t, "cash", got.AccountType)
			return nil
		})
	})

	t.Run("list holdings", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", nil, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"holdings":[{"id":"h1","quantity":10.5,"name":"Vanguard ETF","ticker":"VTI","value":3000.5,"costBasis":2500,"account":{"id":"a1"}},{"id":"h2","quantity":500,"name":"USD","ticker":"CUR:USD","value":500,"costBasis":500,"account":{"id":"a1"}}]}}]}}}`, func(s *Service) error {
			got, err := s.ListHoldings(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 2)
			eq(t, "h1", got[0].ID)
			eq(t, "VTI", got[0].Ticker)
			eq(t, "Vanguard ETF", got[0].Name)
			eq(t, 10.5, got[0].Quantity)
			eq(t, 2500.0, got[0].Basis)
			eq(t, 3000.5, got[0].Value)
			eq(t, "a1", got[0].AccountID)
			eq(t, "CUR:USD", got[1].Ticker)
			return nil
		})
	})

	t.Run("account types", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountTypeOptions", nil, `{"accountTypes":[{"name":"bank"},{"name":"credit"}]}`, func(s *Service) error {
			got, err := s.GetAccountTypes(context.Background())
			mustNoErr(t, err)
			eq(t, []string{"bank", "credit"}, got)
			return nil
		})
	})

	t.Run("refresh status", func(t *testing.T) {
		runGraphQLCase(t, "ForceRefreshAccountsQuery", nil, `{"accounts":[{"id":"acc-1","hasSyncInProgress":false},{"id":"acc-2","hasSyncInProgress":true}]}`, func(s *Service) error {
			got, err := s.GetAccountsRefreshStatus(context.Background())
			mustNoErr(t, err)
			eq(t, false, got["is_complete"])
			eq(t, "syncing", got["status"])
			mustLen(t, got["accounts"], 2)
			return nil
		})
	})
}

func testServiceAccountsCoreMutationPaths(t *testing.T) {
	t.Helper()

	t.Run("create manual account", func(t *testing.T) {
		wantInput := map[string]any{"input": map[string]any{"type": "bank", "subtype": "checking", "includeInNetWorth": true, "name": "Savings", "displayBalance": 10.0}}
		runGraphQLCase(t, "Web_CreateManualAccount", wantInput, `{"createManualAccount":{"account":{"id":"a2","displayName":"Savings","displayBalance":10},"errors":null}}`, func(s *Service) error {
			got, err := s.CreateManualAccount(context.Background(), "Savings", "bank", "checking", 10)
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "a2", got.ID)
			eq(t, "Savings", got.DisplayName)
			eq(t, 10.0, got.DisplayBalance)
			return nil
		})
	})

	t.Run("create manual account payload error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Web_CreateManualAccount")
				return client.respond(result, `{"createManualAccount":{"account":null,"errors":{"message":"subtype is required"}}}`)
			},
		}
		_, err := NewService(client).CreateManualAccount(context.Background(), "Savings", "bank", "", 10)
		if err == nil || !strings.Contains(err.Error(), "subtype is required") {
			t.Fatalf("CreateManualAccount() error = %v, want 'subtype is required'", err)
		}
	})

	t.Run("create manual account payload error list", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Web_CreateManualAccount")
				return client.respond(result, `{"createManualAccount":{"account":null,"errors":[{"message":"subtype is required"},{"message":"other"}]}}`)
			},
		}
		_, err := NewService(client).CreateManualAccount(context.Background(), "Savings", "bank", "", 10)
		if err == nil || !strings.Contains(err.Error(), "subtype is required") {
			t.Fatalf("CreateManualAccount() error = %v, want 'subtype is required'", err)
		}
	})

	t.Run("update account", func(t *testing.T) {
		name := "New name"
		balance := 11.25
		wantInput := map[string]any{"input": map[string]any{"id": "acc-1", "name": name, "displayBalance": balance}}
		runGraphQLCase(t, "Common_UpdateAccount", wantInput, `{"updateAccount":{"account":{"id":"acc-1","displayName":"New name","displayBalance":11.25},"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateAccount(context.Background(), "acc-1", &name, &balance)
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "New name", got.DisplayName)
			eq(t, 11.25, got.DisplayBalance)
			return nil
		})
	})

	t.Run("update account payload error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_UpdateAccount")
				return client.respond(result, `{"updateAccount":{"account":null,"errors":{"message":"account not found"}}}`)
			},
		}
		_, err := NewService(client).UpdateAccount(context.Background(), "acc-1", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "account not found") {
			t.Fatalf("UpdateAccount() error = %v, want 'account not found'", err)
		}
	})

	t.Run("update account payload error list", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_UpdateAccount")
				return client.respond(result, `{"updateAccount":{"account":null,"errors":[{"message":"account not found"}]}}`)
			},
		}
		_, err := NewService(client).UpdateAccount(context.Background(), "acc-1", nil, nil)
		if err == nil || !strings.Contains(err.Error(), "account not found") {
			t.Fatalf("UpdateAccount() error = %v, want 'account not found'", err)
		}
	})

	t.Run("refresh accounts", func(t *testing.T) {
		wantInput := map[string]any{"input": map[string]any{"accountIds": []string{"a1", "a2"}}}
		runGraphQLCase(t, "Common_ForceRefreshAccountsMutation", wantInput, `{"forceRefreshAccounts":{"success":true,"errors":null}}`, func(s *Service) error {
			return s.RefreshAccounts(context.Background(), []string{"a1", "a2"})
		})
	})

	t.Run("refresh accounts not successful", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_ForceRefreshAccountsMutation")
				return client.respond(result, `{"forceRefreshAccounts":{"success":false,"errors":{"message":"refresh unavailable"}}}`)
			},
		}
		err := NewService(client).RefreshAccounts(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "refresh unavailable") {
			t.Fatalf("RefreshAccounts() error = %v, want 'refresh unavailable'", err)
		}
	})

	t.Run("refresh accounts error list", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_ForceRefreshAccountsMutation")
				return client.respond(result, `{"forceRefreshAccounts":{"success":false,"errors":[{"message":"refresh unavailable"}]}}`)
			},
		}
		err := NewService(client).RefreshAccounts(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "refresh unavailable") {
			t.Fatalf("RefreshAccounts() error = %v, want 'refresh unavailable'", err)
		}
	})

	t.Run("refresh accounts blank payload message", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_ForceRefreshAccountsMutation")
				return client.respond(result, `{"forceRefreshAccounts":{"success":false,"errors":{"message":""}}}`)
			},
		}
		err := NewService(client).RefreshAccounts(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "failed to refresh accounts") {
			t.Fatalf("RefreshAccounts() error = %v, want 'failed to refresh accounts'", err)
		}
	})

	t.Run("delete account", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteAccount", map[string]any{"id": "acc-1"}, `{"deleteAccount":{"deleted":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteAccount(context.Background(), "acc-1")
		})
	})

	t.Run("delete account blank payload message", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_DeleteAccount")
				return client.respond(result, `{"deleteAccount":{"deleted":false,"errors":{"message":""}}}`)
			},
		}
		err := NewService(client).DeleteAccount(context.Background(), "acc-1")
		if err == nil || !strings.Contains(err.Error(), "failed to delete account") {
			t.Fatalf("DeleteAccount() error = %v, want 'failed to delete account'", err)
		}
	})

	t.Run("delete account not deleted", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_DeleteAccount")
				return client.respond(result, `{"deleteAccount":{"deleted":false,"errors":{"message":"account not found"}}}`)
			},
		}
		err := NewService(client).DeleteAccount(context.Background(), "acc-1")
		if err == nil || !strings.Contains(err.Error(), "account not found") {
			t.Fatalf("DeleteAccount() error = %v, want 'account not found'", err)
		}
	})

	t.Run("delete account error list", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_DeleteAccount")
				return client.respond(result, `{"deleteAccount":{"deleted":false,"errors":[{"message":"account not found"}]}}`)
			},
		}
		err := NewService(client).DeleteAccount(context.Background(), "acc-1")
		if err == nil || !strings.Contains(err.Error(), "account not found") {
			t.Fatalf("DeleteAccount() error = %v, want 'account not found'", err)
		}
	})
}

func TestPayloadErrorMessage(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`null`, ""},
		{`{"message":"boom"}`, "boom"},
		{`{"message":""}`, ""},
		{`[{"message":"first"},{"message":"second"}]`, "first"},
		{`[{"message":""},{"message":"second"}]`, "second"},
		{`[]`, ""},
		{`garbage`, ""},
	}
	for _, c := range cases {
		if got := payloadErrorMessage(json.RawMessage(c.raw)); got != c.want {
			t.Errorf("payloadErrorMessage(%s) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func testServiceAccountsCoreHistoryPaths(t *testing.T) {
	t.Helper()

	t.Run("account holdings", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", nil, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h1","quantity":2,"basis":3,"totalValue":6,"holdings":[{"id":"h1-1","quantity":2,"name":"Alphabet","ticker":"GOOGL","account":{"id":"acc-1"}}]}},{"node":{"id":"h2","quantity":4,"basis":5,"totalValue":20,"holdings":[{"id":"h2-1","quantity":4,"name":"Other","ticker":"OTR","account":{"id":"acc-2"}}]}}]}}}`, func(s *Service) error {
			got, err := s.GetAccountHoldings(context.Background(), "acc-1")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "h1", got[0].ID)
			eq(t, 3.0, got[0].Basis)
			eq(t, 6.0, got[0].TotalValue)
			return nil
		})
	})

	t.Run("account history", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountHistory", map[string]any{"filters": map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":10}]}`, func(s *Service) error {
			got, err := s.GetAccountHistory(context.Background(), "acc-1", "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, 10.0, got[0].Amount)
			return nil
		})
	})

	t.Run("recent balances", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountRecentBalances", map[string]any{"startDate": "2026-05-01"}, `{"accounts":[{"id":"acc-1","displayName":"Checking","type":{"group":"asset"},"recentBalances":[1,2,3]}]}`, func(s *Service) error {
			got, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Checking", got[0].DisplayName)
			eq(t, "asset", got[0].AccountTypeGroup)
			mustNotNil(t, got[0].RecentBalances)
			return nil
		})
	})

	t.Run("balance at date", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetDisplayBalanceAtDate", map[string]any{"date": "2026-05-10"}, `{"accounts":[{"id":"acc-1","displayName":"Checking","displayBalance":42.25,"type":{"name":"cash","group":"asset"}}]}`, func(s *Service) error {
			got, err := s.GetAccountBalancesAt(context.Background(), "2026-05-10", []string{"acc-1"})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "acc-1", got[0].ID)
			eq(t, "Checking", got[0].DisplayName)
			eq(t, 42.25, got[0].DisplayBalance)
			eq(t, "cash", got[0].AccountType)
			eq(t, "asset", got[0].AccountTypeGroup)
			return nil
		})
	})
}

func testServiceAccountsCoreSnapshotPaths(t *testing.T) {
	t.Helper()

	t.Run("snapshots by type", func(t *testing.T) {
		runGraphQLCase(t, "GetSnapshotsByAccountType", map[string]any{"startDate": "2026-05-01", "timeframe": "month"}, `{"snapshotsByAccountType":[{"accountType":"bank","month":"2026-05","balance":1}],"accountTypes":[{"name":"bank","group":"asset"}]}`, func(s *Service) error {
			got, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month")
			mustNoErr(t, err)
			b, _ := json.Marshal(got)
			hasSubstr(t, string(b), "snapshotsByAccountType")
			return nil
		})
	})

	t.Run("aggregate snapshots", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregateSnapshots", map[string]any{"filters": map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31", "accountType": "bank"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}`, func(s *Service) error {
			got, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "2026-05-31", "bank")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			return nil
		})
	})

	t.Run("aggregate snapshots default filters", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregateSnapshots", map[string]any{"filters": map[string]any{}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}`, func(s *Service) error {
			got, err := s.GetAggregateSnapshots(context.Background(), "", "", "")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			return nil
		})
	})
}

func TestServiceBudgetCashflowAndReferenceMethods(t *testing.T) {
	testServiceBudgetMutationAndReadPaths(t)
	testServiceCashflowAggregationPaths(t)
	testServiceReferenceRulePaths(t)
}

func testServiceBudgetMutationAndReadPaths(t *testing.T) {
	t.Helper()

	t.Run("get budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetJointPlanningData", map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "cat-1", got.CategoryID)
			eq(t, "Food", got.CategoryName)
			eq(t, 100.0, got.Planned)
			eq(t, 80.0, got.Actual)
			return nil
		})
	})

	t.Run("list budgets", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetJointPlanningData", map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.ListBudgets(context.Background(), ListBudgetsOptions{StartDate: "2026-05-01", EndDate: "2026-05-31"})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Food", got[0].CategoryName)
			return nil
		})
	})

	t.Run("set budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateBudgetItem", map[string]any{"input": map[string]any{"categoryId": "cat-1", "amount": 75.5, "timeframe": "month", "startDate": "2026-05-01", "applyToFuture": false}}, `{"updateOrCreateBudgetItem":{"budgetItem":{"id":"bi-1","budgetAmount":75.5}}}`, func(s *Service) error {
			got, err := s.SetBudget(context.Background(), "cat-1", 75.5, "2026-05-01")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "cat-1", got.CategoryID)
			eq(t, 75.5, got.Planned)
			return nil
		})
	})

	t.Run("update flexible budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateFlexBudgetMutation", map[string]any{"input": map[string]any{"month": 5, "year": 2026, "plannedCashFlowAmount": 25.0}}, `{"updateOrCreateFlexBudgetItem":{"flexBudgetItem":{"month":5}}}`, func(s *Service) error {
			return s.UpdateFlexibleBudget(context.Background(), 5, 2026, 25)
		})
	})

	t.Run("rollover settings", func(t *testing.T) {
		runGraphQLCase(t, "UpdateFlexRolloverSettings", map[string]any{"input": map[string]any{"rolloverStartMonth": "may", "rolloverStartingBalance": 100.0, "rolloverEnabled": true}}, `{"updateBudgetSettings":{"budgetRolloverPeriod":{"id":"roll"}}}`, func(s *Service) error {
			return s.UpdateFlexRolloverSettings(context.Background(), "may", 100, true)
		})
	})

	t.Run("reset budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_ResetBudget", map[string]any{"input": map[string]any{"startDate": "2026-05-01", "overwriteExisting": false}}, `{"resetBudget":{"errors":null}}`, func(s *Service) error {
			return s.ResetBudget(context.Background(), "2026-05-01", false, nil)
		})
	})

	t.Run("budget settings", func(t *testing.T) {
		runGraphQLCase(t, "Common_BudgetSettings", nil, `{"budgetSystem":"flex","budgetApplyToFutureMonthsDefault":true,"budgetStatus":{"hasBudget":true,"hasTransactions":true}}`, func(s *Service) error {
			got, err := s.GetBudgetSettings(context.Background())
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "flex", got.System)
			return nil
		})
	})

	t.Run("flex rollover settings", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetFlexibleGroupRolloverSettings", nil, `{"budgetSystem":"flex","flexExpenseRolloverPeriod":{"id":"r-1","startMonth":"2026-01-01","startingBalance":100}}`, func(s *Service) error {
			got, err := s.GetFlexRolloverSettings(context.Background())
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "r-1", got.ID)
			return nil
		})
	})

	t.Run("set group budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateBudgetItem", map[string]any{"input": map[string]any{"categoryGroupId": "g1", "amount": 500.0, "timeframe": "month", "startDate": "2026-05-01", "applyToFuture": false}}, `{"updateOrCreateBudgetItem":{"errors":null}}`, func(s *Service) error {
			return s.SetBudgetGroup(context.Background(), "g1", 500, "2026-05-01")
		})
	})

	t.Run("create budget", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateBudgetForHousehold", map[string]any{"input": map[string]any{"startDate": "2026-05-01", "timeframe": "month"}}, `{"createBudget":{"errors":null}}`, func(s *Service) error {
			return s.CreateBudget(context.Background(), "2026-05-01")
		})
	})

	t.Run("clear budget", func(t *testing.T) {
		runGraphQLCase(t, "Web_ClearAllMutation", map[string]any{"input": map[string]any{"startDate": "2026-05-01"}}, `{"clearBudget":{"errors":null}}`, func(s *Service) error {
			return s.ClearBudget(context.Background(), "2026-05-01")
		})
	})

	t.Run("reset budget rollover", func(t *testing.T) {
		runGraphQLCase(t, "Web_ResetRolloverMutation", map[string]any{"input": map[string]any{"startMonth": "2026-05-01", "categoryId": "c1"}}, `{"resetBudgetRollover":{"errors":null}}`, func(s *Service) error {
			return s.ResetBudgetRollover(context.Background(), &ResetRolloverOptions{StartMonth: "2026-05-01", CategoryID: "c1"})
		})
	})
}

func testServiceCashflowAggregationPaths(t *testing.T) {
	t.Helper()

	t.Run("cashflow", func(t *testing.T) {
		var calls []map[string]any
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "GetTransactionsList")
				calls = append(calls, req.Variables)
				payload := `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-01-01","amount":10,"plaidName":"plaid","merchant":{"name":"Paycheck"},"category":{"name":"Income"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`
				if len(calls) == 2 {
					payload = `{"allTransactions":{"results":[{"id":"tx-2","date":"2026-01-02","amount":-2,"plaidName":"plaid","merchant":{"name":"Transit"},"category":{"name":"Travel"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-3","date":"2026-01-03","amount":-4,"plaidName":"plaid","merchant":{"name":"Coffee"},"category":{"name":"Food"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`
				}
				return client.respond(result, payload)
			},
		}
		got, err := NewService(client).ListCashflow(context.Background(), "2026-01-01", "2026-01-03")
		mustNoErr(t, err)
		mustLen(t, got, 3)
		eq(t, "2026-01-01", got[0].Period)
		eq(t, 10.0, got[0].Income)
		eq(t, -2.0, got[1].Expense)
		eq(t, 0.0, got[1].Income)
		eq(t, -4.0, got[2].Expense)
		mustLen(t, calls, 2)
	})

	t.Run("cashflow summary", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowSummary", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"summary":{"sumIncome":200,"sumExpense":100,"savings":100,"savingsRate":50}}]}`, func(s *Service) error {
			got, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, 50.0, got.SavingsRate)
			return nil
		})
	})

	t.Run("cashflow categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowCategories", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"category":{"name":"Food"}},"summary":{"sum":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Food", got[0].Name)
			return nil
		})
	})

	t.Run("cashflow merchants", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowMerchants", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"merchant":{"name":"Store"}},"summary":{"sumIncome":0,"sumExpense":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Store", got[0].Name)
			return nil
		})
	})

	t.Run("cashflow trends by category group", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregatesGraphCategoryGroup", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-03-31", "categories": []string{"cat-1"}, "accounts": []string{"acc-1"}}}, `{"aggregates":[{"groupBy":{"categoryGroup":{"id":"grp-1"},"month":"2026-01"},"summary":{"sum":-120}}]}`, func(s *Service) error {
			got, err := s.GetCashflowTrends(context.Background(), &CashflowTrendOptions{
				StartDate:   "2026-01-01",
				EndDate:     "2026-03-31",
				GroupBy:     "category-group",
				Period:      "month",
				CategoryIDs: []string{"cat-1"},
				AccountIDs:  []string{"acc-1"},
			})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "grp-1", got[0].GroupID)
			eq(t, "2026-01", got[0].Period)
			eq(t, -120.0, got[0].Sum)
			return nil
		})
	})
}

func testServiceReferenceRulePaths(t *testing.T) {
	t.Helper()

	t.Run("rules list", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionRules", nil, `{"transactionRules":[{"id":"r1","order":1,"merchantCriteriaUseOriginalStatement":false,"merchantCriteria":[{"operator":"contains","value":"coffee"}],"merchantNameCriteria":[{"operator":"eq","value":"shop"}],"amountCriteria":{"operator":"gt","isExpense":true,"value":5,"valueRange":null},"categoryIds":["cat-1"],"accountIds":["acc-1"],"setCategoryAction":{"id":"cat-1","name":"Food"},"setMerchantAction":null,"addTagsAction":[{"id":"tag-1","name":"Trip","color":"blue"}],"linkGoalAction":null,"setHideFromReportsAction":false,"reviewStatusAction":"needs_review","recentApplicationCount":2,"lastAppliedAt":"2026-05-01T00:00:00Z"}]}`, func(s *Service) error {
			got, err := s.ListRules(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			mustLen(t, got[0].MerchantCriteria, 1)
			eq(t, "coffee", got[0].MerchantCriteria[0].Value)
			mustLen(t, got[0].MerchantNameCriteria, 1)
			eq(t, "shop", got[0].MerchantNameCriteria[0].Value)
			return nil
		})
	})

	t.Run("create rule", func(t *testing.T) {
		amount := 5.0
		runGraphQLCase(t, "Common_CreateTransactionRuleMutationV2", map[string]any{"input": map[string]any{"applyToExistingTransactions": false, "merchantNameCriteria": []map[string]any{{"operator": "contains", "value": "coffee"}}, "amountCriteria": map[string]any{"operator": "gt", "isExpense": true, "value": amount, "valueRange": nil}, "setCategoryAction": "cat-1", "accountIds": []string{"acc-1"}}}, `{"createTransactionRuleV2":{"errors":null}}`, func(s *Service) error {
			return s.CreateRule(context.Background(), &CreateRuleInput{RuleFields: RuleFields{
				MerchantOperator: "contains",
				MerchantValue:    "coffee",
				AmountOperator:   "gt",
				AmountValue:      &amount,
				AmountIsExpense:  true,
				SetCategoryID:    "cat-1",
				AccountIDs:       []string{"acc-1"},
			}})
		})
	})

	t.Run("update rule", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateTransactionRuleMutationV2", map[string]any{"input": map[string]any{"id": "r1", "applyToExistingTransactions": true, "merchantNameCriteria": []map[string]any{{"operator": "eq", "value": "shop"}}, "setCategoryAction": "cat-2"}}, `{"updateTransactionRuleV2":{"errors":null}}`, func(s *Service) error {
			return s.UpdateRule(context.Background(), &UpdateRuleInput{
				ID: "r1",
				RuleFields: RuleFields{
					MerchantOperator: "eq",
					MerchantValue:    "shop",
					SetCategoryID:    "cat-2",
					ApplyToExisting:  true,
				},
			})
		})
	})

	t.Run("create rule full parity", func(t *testing.T) {
		lower := 10.0
		upper := 50.0
		useOrig := true
		ownerJoint := true
		bizUnassigned := true
		hide := true
		notify := true
		paydown := true
		actionJoint := true
		actionBizUnassigned := true
		want := map[string]any{"input": map[string]any{
			"merchantNameCriteria":                 []map[string]any{{"operator": "contains", "value": "Uber"}},
			"merchantCriteria":                     []map[string]any{{"operator": "eq", "value": "Legacy Co"}},
			"merchantCriteriaUseOriginalStatement": true,
			"originalStatementCriteria":            []map[string]any{{"operator": "contains", "value": "UBER *TRIP"}},
			"amountCriteria":                       map[string]any{"operator": "between", "isExpense": true, "value": lower, "valueRange": map[string]any{"lower": lower, "upper": upper}},
			"categoryIds":                          []string{"cat-1"},
			"accountIds":                           []string{"acc-1"},
			"criteriaOwnerUserIds":                 []string{"u-1"},
			"criteriaOwnerIsJoint":                 true,
			"criteriaBusinessEntityIds":            []string{"b-1"},
			"criteriaBusinessEntityIsUnassigned":   true,
			"setCategoryAction":                    "cat-9",
			"setMerchantAction":                    "Uber Rides",
			"addTagsAction":                        []string{"tag-1"},
			"setHideFromReportsAction":             true,
			"reviewStatusAction":                   "needs_review",
			"needsReviewByUserAction":              "u-2",
			"linkGoalAction":                       "goal-1",
			"linkSavingsGoalAction":                "sgoal-1",
			"setLinkToPaydownBudgetAction":         true,
			"sendNotificationAction":               true,
			"actionSetOwner":                       "u-3",
			"actionSetOwnerIsJoint":                true,
			"actionSetBusinessEntity":              "b-2",
			"actionSetBusinessEntityIsUnassigned":  true,
			"splitTransactionsAction":              map[string]any{"amountType": "PERCENTAGE", "splitsInfo": []map[string]any{{"amount": 0.6, "categoryId": "cat-1"}}},
			"applyToExistingTransactions":          true,
		}}
		runGraphQLCase(t, "Common_CreateTransactionRuleMutationV2", want, `{"createTransactionRuleV2":{"errors":null}}`, func(s *Service) error {
			return s.CreateRule(context.Background(), &CreateRuleInput{RuleFields: RuleFields{
				MerchantOperator: "contains", MerchantValue: "Uber",
				LegacyMerchantOperator: "eq", LegacyMerchantValue: "Legacy Co",
				UseOriginalStatement:                &useOrig,
				OriginalStatementOperator:           "contains",
				OriginalStatementValue:              "UBER *TRIP",
				AmountOperator:                      "between",
				AmountValue:                         &lower,
				AmountValueUpper:                    &upper,
				AmountIsExpense:                     true,
				CategoryIDs:                         []string{"cat-1"},
				AccountIDs:                          []string{"acc-1"},
				CriteriaOwnerUserIDs:                []string{"u-1"},
				CriteriaOwnerIsJoint:                &ownerJoint,
				CriteriaBusinessEntityIDs:           []string{"b-1"},
				CriteriaBusinessEntityIsUnassigned:  &bizUnassigned,
				SetCategoryID:                       "cat-9",
				SetMerchant:                         "Uber Rides",
				AddTagIDs:                           []string{"tag-1"},
				HideFromReports:                     &hide,
				ReviewStatus:                        "needs_review",
				NeedsReviewByUserID:                 "u-2",
				LinkGoalID:                          "goal-1",
				LinkSavingsGoalID:                   "sgoal-1",
				LinkToPaydownBudget:                 &paydown,
				SendNotification:                    &notify,
				ActionSetOwner:                      "u-3",
				ActionSetOwnerIsJoint:               &actionJoint,
				ActionSetBusinessEntity:             "b-2",
				ActionSetBusinessEntityIsUnassigned: &actionBizUnassigned,
				SplitAction: &RuleSplitAction{
					AmountType: "PERCENTAGE",
					SplitsInfo: []map[string]any{{"amount": 0.6, "categoryId": "cat-1"}},
				},
				ApplyToExisting: true,
			}})
		})
	})

	t.Run("update rule full parity", func(t *testing.T) {
		hide := false
		want := map[string]any{"input": map[string]any{
			"id":                          "r1",
			"setMerchantAction":           "Lyft Rides",
			"reviewStatusAction":          "reviewed",
			"setHideFromReportsAction":    false,
			"applyToExistingTransactions": false,
		}}
		runGraphQLCase(t, "Common_UpdateTransactionRuleMutationV2", want, `{"updateTransactionRuleV2":{"errors":null}}`, func(s *Service) error {
			return s.UpdateRule(context.Background(), &UpdateRuleInput{
				ID: "r1",
				RuleFields: RuleFields{
					SetMerchant:     "Lyft Rides",
					ReviewStatus:    "reviewed",
					HideFromReports: &hide,
				},
			})
		})
	})

	t.Run("rules list full parity", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionRules", nil, `{"transactionRules":[{"id":"r1","order":1,"merchantCriteriaUseOriginalStatement":true,"merchantCriteria":[{"operator":"eq","value":"Legacy"}],"merchantNameCriteria":[{"operator":"contains","value":"Uber"}],"originalStatementCriteria":[{"operator":"contains","value":"UBER"}],"amountCriteria":{"operator":"between","isExpense":true,"value":10,"valueRange":{"lower":10,"upper":50}},"categoryIds":["cat-1"],"categories":[{"id":"cat-1","name":"Food"}],"accountIds":["acc-1"],"accounts":[{"id":"acc-1","displayName":"Checking"}],"criteriaOwnerIsJoint":true,"criteriaOwnerUserIds":["u-1"],"criteriaOwnerUsers":[{"id":"u-1","displayName":"Ann"}],"criteriaBusinessEntityIds":["b-1"],"criteriaBusinessEntityIsUnassigned":false,"criteriaBusinessEntities":[{"id":"b-1","name":"Biz"}],"setCategoryAction":{"id":"cat-9","name":"Transport"},"setMerchantAction":{"id":"m-9","name":"Uber Rides"},"addTagsAction":[{"id":"tag-1","name":"Trip","color":"blue"}],"linkGoalAction":{"id":"goal-1","name":"Trip"},"linkSavingsGoalAction":{"id":"sgoal-1","name":"Save"},"needsReviewByUserAction":{"id":"u-2","displayName":"Bob"},"unassignNeedsReviewByUserAction":false,"sendNotificationAction":true,"setHideFromReportsAction":true,"setLinkToPaydownBudgetAction":false,"reviewStatusAction":"needs_review","actionSetOwnerIsJoint":false,"actionSetOwner":{"id":"u-3","displayName":"Cat"},"actionSetBusinessEntity":{"id":"b-2","name":"Biz2"},"actionSetBusinessEntityIsUnassigned":false,"splitTransactionsAction":{"amountType":"PERCENTAGE","splitsInfo":[{"amount":0.6,"categoryId":"cat-1"}]},"recentApplicationCount":2,"lastAppliedAt":"2026-05-01T00:00:00Z"}]}`, func(s *Service) error {
			got, err := s.ListRules(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			r := got[0]
			eq(t, true, r.MerchantCriteriaUseOriginalStatement)
			mustLen(t, r.MerchantCriteria, 1)
			mustLen(t, r.OriginalStatementCriteria, 1)
			eq(t, "UBER", r.OriginalStatementCriteria[0].Value)
			mustNotNil(t, r.AmountCriteria)
			mustNotNil(t, r.AmountCriteria.ValueRange)
			eq(t, 10.0, r.AmountCriteria.ValueRange.Lower)
			eq(t, 50.0, r.AmountCriteria.ValueRange.Upper)
			mustLen(t, r.Categories, 1)
			eq(t, "Food", r.Categories[0].Name)
			mustLen(t, r.Accounts, 1)
			eq(t, "Checking", r.Accounts[0].Name)
			mustLen(t, r.CriteriaOwnerUsers, 1)
			mustLen(t, r.CriteriaBusinessEntities, 1)
			eq(t, "Biz", r.CriteriaBusinessEntities[0].Name)
			mustNotNil(t, r.SetMerchantAction)
			eq(t, "Uber Rides", r.SetMerchantAction.Name)
			mustNotNil(t, r.ReviewStatusAction)
			eq(t, "needs_review", *r.ReviewStatusAction)
			mustNotNil(t, r.NeedsReviewByUserAction)
			eq(t, "u-2", r.NeedsReviewByUserAction.ID)
			mustNotNil(t, r.LinkGoalAction)
			mustNotNil(t, r.LinkSavingsGoalAction)
			mustNotNil(t, r.SendNotificationAction)
			mustNotNil(t, r.SetHideFromReportsAction)
			mustNotNil(t, r.SetLinkToPaydownBudgetAction)
			mustNotNil(t, r.ActionSetOwner)
			eq(t, "u-3", r.ActionSetOwner.ID)
			mustNotNil(t, r.ActionSetBusinessEntity)
			mustNotNil(t, r.SplitTransactionsAction)
			eq(t, "PERCENTAGE", r.SplitTransactionsAction.AmountType)
			mustLen(t, r.SplitTransactionsAction.SplitsInfo, 1)
			return nil
		})
	})

	t.Run("delete rule", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteTransactionRule", map[string]any{"id": "r1"}, `{"deleteTransactionRule":{"deleted":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteRule(context.Background(), "r1")
		})
	})

	t.Run("delete rule not deleted", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_DeleteTransactionRule")
				return client.respond(result, `{"deleteTransactionRule":{"deleted":false,"errors":{"message":"not found"}}}`)
			},
		}
		err := NewService(client).DeleteRule(context.Background(), "r1")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("DeleteRule() error = %v, want 'not found'", err)
		}
	})
}

func TestServiceTagsCategoriesAndLookupMethods(t *testing.T) {
	testServiceTagAndCategoryCRUDPaths(t)
	testServiceLookupAndSubscriptionPaths(t)
	testServiceRecurringAndGoalsPaths(t)
}

func testServiceTagAndCategoryCRUDPaths(t *testing.T) {
	t.Helper()

	t.Run("list tags", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, `{"householdTransactionTags":[{"id":"tag-1","name":"Trip","color":"blue","order":2}]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background(), "", 0)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Trip", got[0].Name)
			eq(t, 2, got[0].Order)
			return nil
		})
	})

	t.Run("create tag", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionTag", map[string]any{"input": map[string]any{"name": "Trip", "color": "blue"}}, `{"createTransactionTag":{"tag":{"id":"tag-1","name":"Trip","color":"blue","order":1},"errors":null}}`, func(s *Service) error {
			got, err := s.CreateTag(context.Background(), "Trip", "blue")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "tag-1", got.ID)
			eq(t, "Trip", got.Name)
			return nil
		})
	})

	t.Run("get tag", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, `{"householdTransactionTags":[{"id":"tag-1","name":"Trip","color":"blue","order":1}]}`, func(s *Service) error {
			got, err := s.GetTag(context.Background(), "tag-1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "Trip", got.Name)
			return nil
		})
	})

	t.Run("get tag missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, `{"householdTransactionTags":[]}`, func(s *Service) error {
			_, err := s.GetTag(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete tag", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHouseholdTransactionTag", map[string]any{"tagId": "tag-1"}, `{"deleteTransactionTag":{"errors":null}}`, func(s *Service) error {
			return s.DeleteTag(context.Background(), "tag-1")
		})
	})

	t.Run("reorder tag", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateTransactionTagOrder", map[string]any{"tagId": "tag-1", "order": 2}, `{"updateTransactionTagOrder":{"householdTransactionTags":[{"id":"tag-1","name":"Trip","color":"blue","order":2}]}}`, func(s *Service) error {
			got, err := s.ReorderTag(context.Background(), "tag-1", 2)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, 2, got[0].Order)
			return nil
		})
	})

	t.Run("list category groups", func(t *testing.T) {
		runGraphQLCase(t, "ManageGetCategoryGroups", nil, `{"categoryGroups":[{"id":"g1","name":"Income","type":"income","categories":[{"id":"c1","name":"Salary"}]}]}`, func(s *Service) error {
			got, err := s.ListCategoryGroups(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			mustLen(t, got[0].Categories, 1)
			eq(t, "Salary", got[0].Categories[0].Name)
			return nil
		})
	})

	t.Run("list categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCategories", nil, `{"categories":[{"id":"c1","name":"Food","order":2,"icon":"restaurant","group":{"id":"g1","name":"Expenses","type":"expense"}}]}`, func(s *Service) error {
			got, err := s.ListCategories(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Expenses", got[0].GroupName)
			eq(t, "g1", got[0].GroupID)
			eq(t, "expense", got[0].GroupType)
			eq(t, "restaurant", got[0].Icon)
			eq(t, 2, got[0].Order)
			return nil
		})
	})

	t.Run("create category", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateCategory", map[string]any{"input": map[string]any{"name": "Food", "group": "g1"}}, `{"createCategory":{"errors":null,"category":{"id":"c1","order":1,"name":"Food","icon":"cart","group":{"id":"g1","name":"Expenses","type":"expense"}}}}`, func(s *Service) error {
			got, err := s.CreateCategory(context.Background(), "Food", "g1", "")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "c1", got.ID)
			eq(t, "Food", got.Name)
			eq(t, "g1", got.GroupID)
			return nil
		})
	})

	t.Run("delete category", func(t *testing.T) {
		runGraphQLCase(t, "Web_DeleteCategory", map[string]any{"id": "c1"}, `{"deleteCategory":{"errors":null,"deleted":true}}`, func(s *Service) error {
			return s.DeleteCategory(context.Background(), "c1", "")
		})
	})

	t.Run("show category", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetEditCategory", map[string]any{"id": "c1"}, `{"category":{"id":"c1","order":1,"name":"Food","icon":"cart","group":{"id":"g1","name":"Expenses","type":"expense"}}}`, func(s *Service) error {
			got, err := s.GetCategory(context.Background(), "c1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "Food", got.Name)
			return nil
		})
	})

	t.Run("reactivate category", func(t *testing.T) {
		runGraphQLCase(t, "Web_RestoreCategory", map[string]any{"id": "c1"}, `{"restoreCategory":{"errors":null,"category":{"id":"c1","order":1,"name":"Food","icon":"cart","group":{"id":"g1","name":"Expenses","type":"expense"}}}}`, func(s *Service) error {
			got, err := s.ReactivateCategory(context.Background(), "c1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "c1", got.ID)
			return nil
		})
	})

	t.Run("reorder category", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateCategoryOrder", map[string]any{"id": "c1", "categoryGroupId": "g1", "order": 2}, `{"updateCategoryOrderInCategoryGroup":{"category":{"id":"c1","order":2,"name":"Food","icon":"cart","group":{"id":"g1","name":"Expenses","type":"expense"}}}}`, func(s *Service) error {
			got, err := s.ReorderCategory(context.Background(), "c1", "g1", 2)
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, 2, got.Order)
			return nil
		})
	})

	t.Run("create category group", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateCategoryGroup", map[string]any{"input": map[string]any{"name": "Pets", "type": "expense"}}, `{"createCategoryGroup":{"categoryGroup":{"id":"g9","name":"Pets","order":9,"type":"expense"}}}`, func(s *Service) error {
			got, err := s.CreateCategoryGroup(context.Background(), "Pets", "expense")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "g9", got.ID)
			return nil
		})
	})

	t.Run("delete category group", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteCategoryGroup", map[string]any{"id": "g9"}, `{"deleteCategoryGroup":{"deleted":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteCategoryGroup(context.Background(), "g9", "")
		})
	})

	t.Run("reorder category group", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateCategoryGroupOrder", map[string]any{"id": "g9", "order": 1}, `{"updateCategoryGroupOrder":{"categoryGroups":[{"id":"g9","name":"Pets","order":1,"type":"expense"}]}}`, func(s *Service) error {
			got, err := s.ReorderCategoryGroup(context.Background(), "g9", 1)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "g9", got[0].ID)
			return nil
		})
	})

	t.Run("delete categories", func(t *testing.T) {
		runGraphQLCase(t, "DeleteCategories", map[string]any{"ids": []string{"c1", "c2"}}, `{"deleteTransactionCategories":{"ok":true}}`, func(s *Service) error {
			return s.DeleteCategories(context.Background(), []string{"c1", "c2"})
		})
	})

	t.Run("update category", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateCategory", map[string]any{"input": map[string]any{"id": "c1", "name": "Groceries"}}, `{"updateCategory":{"errors":[],"category":{"id":"c1","name":"Groceries","icon":"cart","budgetVariability":"flexible","excludeFromBudget":false,"group":{"id":"g1","type":"expense"}}}}`, func(s *Service) error {
			name := "Groceries"
			got, err := s.UpdateCategory(context.Background(), "c1", UpdateCategoryOptions{Name: &name})
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "Groceries", got.Name)
			eq(t, "cart", got.Icon)
			return nil
		})
	})

	t.Run("get category rollover", func(t *testing.T) {
		runGraphQLCase(t, "GetCategoryRollover", map[string]any{"id": "c1"}, `{"category":{"id":"c1","name":"Food","rolloverPeriod":{"id":"rp1","startMonth":"2026-01-01","startingBalance":100,"type":"monthly","frequency":"monthly","targetAmount":500}}}`, func(s *Service) error {
			got, err := s.GetCategoryRollover(context.Background(), "c1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "Food", got.Name)
			eq(t, "2026-01-01", got.StartMonth)
			eq(t, 100.0, got.StartingBalance)
			return nil
		})
	})

	t.Run("update category group", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateCategoryGroup", map[string]any{"input": map[string]any{"id": "g1", "name": "Food & Drink"}}, `{"updateCategoryGroup":{"categoryGroup":{"id":"g1","name":"Food & Drink","order":1,"type":"expense","color":"","groupLevelBudgetingEnabled":false,"budgetVariability":"fixed"}}}`, func(s *Service) error {
			name := "Food & Drink"
			got, err := s.UpdateCategoryGroup(context.Background(), "g1", UpdateCategoryGroupOptions{Name: &name})
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "Food & Drink", got.Name)
			return nil
		})
	})
}

func testServiceLookupAndSubscriptionPaths(t *testing.T) {
	t.Helper()

	t.Run("credit history", func(t *testing.T) {
		runGraphQLCase(t, "GetCreditScoreSnapshots", nil, `{"creditScoreSnapshots":[{"reportedDate":"2026-05-01","score":790,"user":{"id":"u-1"}}]}`, func(s *Service) error {
			got, err := s.GetCreditHistory(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, 790, got[0].Score)
			eq(t, "u-1", got[0].UserID)
			return nil
		})
	})

	t.Run("institutions", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetInstitutionSettings", nil, `{"credentials":[{"id":"c1","updateRequired":false,"disconnectedFromDataProviderAt":"","dataProvider":"plaid","institution":{"id":"i1","plaidInstitutionId":"https://bank.example","name":"Bank","status":"connected"}}]}`, func(s *Service) error {
			got, err := s.ListInstitutions(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "https://bank.example", got[0].URL)
			return nil
		})
	})

	t.Run("subscription", func(t *testing.T) {
		runGraphQLCase(t, "GetSubscriptionDetails", nil, `{"subscription":{"id":"sub-1","paymentSource":"card","referralCode":"REF","isOnFreeTrial":true,"hasPremiumEntitlement":true}}`, func(s *Service) error {
			got, err := s.GetSubscriptionDetails(context.Background())
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "card", got.PaymentSource)
			eq(t, "REF", got.ReferralCode)
			isTrue(t, got.IsOnFreeTrial)
			isTrue(t, got.HasPremiumEntitlement)
			return nil
		})
	})
}

func testServiceRecurringAndGoalsPaths(t *testing.T) {
	t.Helper()

	t.Run("recurring list", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]any{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]any{}}, `{"recurringTransactionItems":[{"stream":{"id":"r1","frequency":"monthly","amount":20,"isApproximate":false,"merchant":{"name":"Gym"}},"date":"2026-06-01","isPast":false,"transactionId":"tx-1","amount":20,"amountDiff":0,"category":{"id":"c1","name":"Food"},"account":{"id":"acc-1","displayName":"Checking"}}]}`, func(s *Service) error {
			got, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Gym", got[0].Merchant)
			return nil
		})
	})

	t.Run("recurring item details preserve stream and item fields", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]any{"startDate": "2025-05-01", "endDate": "2026-05-31", "filters": map[string]any{}}, `{"recurringTransactionItems":[{"stream":{"id":"r1","frequency":"yearly","amount":120,"isApproximate":true,"merchant":{"name":"Cloud Box"}},"date":"2026-02-01","isPast":true,"transactionId":"tx-1","amount":120,"amountDiff":5,"category":{"id":"c1","name":"Software"},"account":{"id":"acc-1","displayName":"Checking"}}]}`, func(s *Service) error {
			got, err := s.ListRecurringItems(context.Background(), "2025-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "r1", got[0].Stream.ID)
			eq(t, "yearly", got[0].Stream.Frequency)
			eq(t, true, got[0].Stream.IsApproximate)
			eq(t, "Cloud Box", got[0].Stream.MerchantName)
			eq(t, "Software", got[0].CategoryName)
			eq(t, "acc-1", got[0].AccountID)
			eq(t, "Checking", got[0].AccountName)
			return nil
		})
	})

	t.Run("recurring update", func(t *testing.T) {
		runGraphQLCase(t, "UpdateRecurringTransaction", map[string]any{"id": "r1", "amount": 21.5}, `{"updateRecurringTransaction":{"recurringTransaction":{"id":"r1","amount":21.5}}}`, func(s *Service) error {
			got, err := s.UpdateRecurring(context.Background(), "r1", 21.5)
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "r1", got.ID)
			eq(t, 21.5, got.Amount)
			return nil
		})
	})

	t.Run("goals list", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoals", nil, `{"savingsGoals":[{"id":"goal-1","name":"Vacation","type":"savings_goal","status":"active","progress":0.75,"currentBalance":7500,"targetAmount":10000,"plannedMonthlyContribution":500,"isSinkingFund":false,"priority":1}]}`, func(s *Service) error {
			got, err := s.ListGoals(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "goal-1", got[0].ID)
			eq(t, "Vacation", got[0].Name)
			eq(t, "active", got[0].Status)
			eq(t, 7500.0, got[0].CurrentBalance)
			eq(t, 10000.0, got[0].TargetAmount)
			return nil
		})
	})

	t.Run("savings goal budgets", func(t *testing.T) {
		runGraphQLCase(t, "GetSavingsGoals", map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"savingsGoalMonthlyBudgetAmounts":[{"id":"sgb-1","savingsGoal":{"id":"goal-1","name":"Vacation","type":"savings_goal","status":"active"},"monthlyAmounts":[{"month":"2026-05","plannedAmount":500,"actualAmount":450,"remainingAmount":50}]}]}`, func(s *Service) error {
			got, err := s.ListSavingsGoalBudgets(context.Background(), "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Vacation", got[0].GoalName)
			eq(t, 500.0, got[0].Planned)
			eq(t, 450.0, got[0].Actual)
			return nil
		})
	})
}

func TestServiceTransactionMethods(t *testing.T) {
	testServiceTransactionReadPaths(t)
	testServiceTransactionMutationPaths(t)
	testServiceTransactionListingPaths(t)
}

func testServiceTransactionReadPaths(t *testing.T) {
	t.Helper()

	t.Run("get transaction", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionDrawer", map[string]any{"id": "tx-1"}, `{"getTransaction":{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","account":{"id":"acc-1","displayName":"Checking"},"tags":[{"id":"tag-1","name":"Trip","color":"blue"}]}}`, func(s *Service) error {
			got, err := s.GetTransaction(context.Background(), "tx-1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			mustLen(t, got.Tags, 1)
			eq(t, "Store", got.Merchant)
			eq(t, "Trip", got.Tags[0].Name)
			eq(t, "acc-1", got.AccountID)
			return nil
		})
	})

	t.Run("transactions summary", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionsPage", map[string]any{"filters": map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregates":[{"summary":{"avg":20,"count":1,"max":20,"maxExpense":20,"sum":20,"sumIncome":0,"sumExpense":20,"first":"2026-05-01","last":"2026-05-31"}}]}`, func(s *Service) error {
			got, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, 1, got.Count)
			eq(t, 20.0, got.Sum)
			eq(t, "2026-05-01", got.First)
			return nil
		})
	})

	t.Run("duplicate transactions", func(t *testing.T) {
		var client *mockClient
		callCount := 0
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "GetTransactionsList")
				expectVars(t, req.Variables, map[string]any{"offset": 0, "limit": 1000, "filters": map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}})
				callCount++
				return client.respond(result, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-2","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-3","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`)
			},
		}
		got, err := NewService(client).GetDuplicateTransactions(context.Background(), "2026-05-01", "2026-05-31")
		mustNoErr(t, err)
		eq(t, 1, callCount)
		mustLen(t, got, 3)
	})

	t.Run("get transaction splits", func(t *testing.T) {
		runGraphQLCase(t, "TransactionSplitQuery", map[string]any{"id": "tx-1"}, `{"getTransaction":{"id":"tx-1","amount":-100,"splitTransactions":[{"id":"s1","amount":-60,"notes":"groceries","merchant":{"name":"Store"},"category":{"name":"Food"}},{"id":"s2","amount":-40,"notes":"household","merchant":{"name":"Store"},"category":{"name":"Home"}}]}}`, func(s *Service) error {
			got, err := s.GetTransactionSplits(context.Background(), "tx-1")
			mustNoErr(t, err)
			mustLen(t, got, 2)
			eq(t, -60.0, got[0].Amount)
			eq(t, "Home", got[1].Category)
			return nil
		})
	})
}

func testServiceTransactionMutationPaths(t *testing.T) {
	t.Helper()

	t.Run("update transaction", func(t *testing.T) {
		notes := "updated"
		categoryID := "cat-1"
		runGraphQLCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]any{"input": map[string]any{"id": "tx-1", "notes": notes, "category": categoryID}}, `{"updateTransaction":{"transaction":{"id":"tx-1","amount":0,"date":"","notes":"updated","hideFromReports":false,"needsReview":false,"category":{"name":"Food"},"merchant":{"name":""}}}}`, func(s *Service) error {
			got, err := s.UpdateTransaction(context.Background(), "tx-1", &notes, &categoryID, nil, nil, nil, nil, nil)
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "updated", got.Notes)
			eq(t, "Food", got.Category)
			return nil
		})
	})

	t.Run("delete transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteTransactionMutation", map[string]any{"input": map[string]any{"transactionId": "tx-1"}}, `{"deleteTransaction":{"ok":true}}`, func(s *Service) error {
			return s.DeleteTransaction(context.Background(), "tx-1")
		})
	})

	t.Run("update splits", func(t *testing.T) {
		runGraphQLCase(t, "Common_SplitTransactionMutation", map[string]any{"input": map[string]any{"transactionId": "tx-1", "splitData": []map[string]any{{"amount": 10.0, "categoryId": "cat-1", "notes": "split"}}}}, `{"updateTransactionSplit":{"errors":[],"transaction":{"id":"tx-1","hasSplitTransactions":true,"splitTransactions":[]}}}`, func(s *Service) error {
			return s.UpdateTransactionSplits(context.Background(), "tx-1", []SplitInput{{Amount: 10, CategoryID: "cat-1", Notes: "split"}})
		})
	})

	t.Run("unsplit transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_SplitTransactionMutation", map[string]any{"input": map[string]any{"transactionId": "tx-1", "splitData": []map[string]any{}}}, `{"updateTransactionSplit":{"errors":[],"transaction":{"id":"tx-1"}}}`, func(s *Service) error {
			return s.UnsplitTransaction(context.Background(), "tx-1")
		})
	})

	t.Run("link transaction to goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_LinkTransactionToGoal", map[string]any{"input": map[string]any{"transactionId": "tx-1", "goalId": "goal-1", "accountId": "acc-1"}}, `{"linkTransactionToGoal":{"goalEvent":{"id":"ge-1"},"errors":null}}`, func(s *Service) error {
			return s.LinkTransactionToGoal(context.Background(), "tx-1", "goal-1", "acc-1")
		})
	})

	t.Run("unlink transaction goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_LinkTransactionToGoal", map[string]any{"input": map[string]any{"transactionId": "tx-1"}}, `{"linkTransactionToGoal":{"goalEvent":null,"errors":null}}`, func(s *Service) error {
			return s.LinkTransactionToGoal(context.Background(), "tx-1", "", "")
		})
	})

	t.Run("get transaction attachment", func(t *testing.T) {
		runGraphQLCase(t, "Mobile_GetAttachmentDetails", map[string]any{"attachmentId": "att-1"}, `{"transactionAttachment":{"id":"att-1","extension":"pdf","filename":"receipt.pdf","originalAssetUrl":"https://example.com/receipt.pdf","sizeBytes":1024}}`, func(s *Service) error {
			got, err := s.GetTransactionAttachment(context.Background(), "att-1")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "receipt.pdf", got.Filename)
			return nil
		})
	})

	t.Run("delete transaction attachment", func(t *testing.T) {
		runGraphQLCase(t, "Web_TransactionDrawerDeleteAttachment", map[string]any{"id": "att-1"}, `{"deleteTransactionAttachment":{"deleted":true}}`, func(s *Service) error {
			return s.DeleteTransactionAttachment(context.Background(), "att-1")
		})
	})

	t.Run("create transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionMutation", map[string]any{"input": map[string]any{"date": "2026-05-08", "accountId": "acc-1", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "lunch", "shouldUpdateBalance": false}}, `{"createTransaction":{"transaction":{"id":"tx-1","amount":-20,"date":"2026-05-08","merchant":{"name":"Store"}}}}`, func(s *Service) error {
			got, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "acc-1", "lunch")
			mustNoErr(t, err)
			mustNotNil(t, got)
			eq(t, "tx-1", got.ID)
			eq(t, "Store", got.Merchant)
			return nil
		})
	})

	t.Run("set transaction tags", func(t *testing.T) {
		runGraphQLCase(t, "Web_SetTransactionTags", map[string]any{"input": map[string]any{"transactionId": "tx-1", "tagIds": []string{"tag-1", "tag-2"}}}, `{"setTransactionTags":{"ok":true}}`, func(s *Service) error {
			return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1", "tag-2"})
		})
	})
}

func testServiceTransactionListingPaths(t *testing.T) {
	t.Helper()

	t.Run("list transactions", func(t *testing.T) {
		pending := false
		hideFromReports := true
		runGraphQLCase(t, "GetTransactionsList", map[string]any{"limit": 50, "offset": 5, "filters": map[string]any{"search": "lunch", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "goals": []string{"goal-1"}, "startDate": "2026-05-01", "endDate": "2026-05-31", "isPending": false, "hideFromReports": true}}, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"pending":false,"hideFromReports":true,"dataProviderDescription":"STORE 123","plaidName":"plaid","merchant":{"name":"Store"},"category":{"name":"Food","group":{"id":"grp-1","name":"Dining","type":"expense"}},"account":{"id":"acc-1","displayName":"Checking","order":4,"type":{"group":"asset"}},"ownedByUser":{"displayName":"Alex"},"goal":{"id":"goal-1","name":"Vacation"},"notes":"lunch"}],"totalCount":1}}`, func(s *Service) error {
			got, total, err := s.ListTransactions(context.Background(), &ListTransactionsOptions{Limit: 50, Offset: 5, Search: "lunch", StartDate: "2026-05-01", EndDate: "2026-05-31", Pending: &pending, HideFromReports: &hideFromReports, GoalIDs: []string{"goal-1"}})
			mustNoErr(t, err)
			eq(t, 1, total)
			mustLen(t, got, 1)
			eq(t, "Food", got[0].Category)
			eq(t, "acc-1", got[0].AccountID)
			eq(t, "goal-1", got[0].Goal.ID)
			eq(t, "Alex", got[0].OwnerDisplayName)
			eq(t, 4, got[0].AccountOrder)
			eq(t, "asset", got[0].AccountTypeGroup)
			eq(t, "grp-1", got[0].CategoryGroup.ID)
			eq(t, "STORE 123", got[0].DataProviderDescription)
			return nil
		})
	})

	t.Run("list all transactions pages until total reached", func(t *testing.T) {
		var calls []int
		client := &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "GetTransactionsList")
				offset, _ := req.Variables["offset"].(int)
				calls = append(calls, offset)
				filters, _ := req.Variables["filters"].(map[string]any)
				if filters["startDate"] != "2026-05-01" || filters["endDate"] != "2026-05-31" {
					t.Fatalf("filters = %#v", filters)
				}
				switch offset {
				case 0:
					return clientRespond(result, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-01","amount":-10,"merchant":{"name":"A"},"category":{"name":"Food"},"account":{"id":"acc-1"}}],"totalCount":2}}`)
				case 1:
					return clientRespond(result, `{"allTransactions":{"results":[{"id":"tx-2","date":"2026-05-02","amount":-20,"merchant":{"name":"B"},"category":{"name":"Food"},"account":{"id":"acc-1"}}],"totalCount":2}}`)
				default:
					t.Fatalf("unexpected offset %d", offset)
				}
				return nil
			},
		}
		got, err := NewService(client).ListAllTransactions(context.Background(), &ListTransactionsOptions{Limit: 1, StartDate: "2026-05-01", EndDate: "2026-05-31"})
		mustNoErr(t, err)
		mustLen(t, got, 2)
		eq(t, []int{0, 1}, calls)
	})
}

func TestServiceInvestments(t *testing.T) {
	t.Run("portfolio", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetPortfolio", map[string]any{"portfolioInput": map[string]any{"startDate": "2026-01-01", "endDate": "2026-05-10", "accounts": []string{"acc-1"}}}, `{"portfolio":{"performance":{"totalValue":1000,"totalChangePercent":0.12,"totalChangeDollars":120},"aggregateHoldings":{"edges":[{"node":{"id":"node-1","quantity":2,"basis":400,"totalValue":1000,"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund","currentPrice":500},"holdings":[{"id":"hold-1","type":"equity","typeDisplay":"Equity","name":"ABC Fund","ticker":"ABC","quantity":2,"value":1000,"account":{"id":"acc-1","displayName":"Brokerage","type":{"name":"investment","display":"Investment"},"subtype":{"name":"brokerage","display":"Brokerage"}}}]}}]}}}`, func(s *Service) error {
			got, err := s.GetInvestmentPortfolio(context.Background(), InvestmentPortfolioOptions{StartDate: "2026-01-01", EndDate: "2026-05-10", AccountIDs: []string{"acc-1"}})
			mustNoErr(t, err)
			eq(t, 1000.0, got.Performance.TotalValue)
			mustLen(t, got.Holdings, 1)
			eq(t, "node-1", got.Holdings[0].ID)
			eq(t, "sec-1", got.Holdings[0].Security.ID)
			mustLen(t, got.Holdings[0].Holdings, 1)
			eq(t, "Brokerage", got.Holdings[0].Holdings[0].Account.DisplayName)
			return nil
		})
	})

	t.Run("security performance", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetInvestmentsHoldingDrawerHistoricalPerformance", map[string]any{"input": map[string]any{"securityIds": []string{"sec-1"}, "startDate": "2026-01-01", "endDate": "2026-05-10"}}, `{"securityHistoricalPerformance":[{"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund"},"historicalChart":[{"date":"2026-01-01","returnPercent":0.1,"value":100}]}]}`, func(s *Service) error {
			got, err := s.GetSecurityPerformance(context.Background(), SecurityPerformanceOptions{SecurityIDs: []string{"sec-1"}, StartDate: "2026-01-01", EndDate: "2026-05-10", IncludeValues: true})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "ABC", got[0].Security.Ticker)
			mustLen(t, got[0].HistoricalChart, 1)
			mustNotNil(t, got[0].HistoricalChart[0].Value)
			eq(t, 100.0, *got[0].HistoricalChart[0].Value)
			return nil
		})
	})
}

func TestServiceCacheAndExportHelpers(t *testing.T) {
	t.Run("export csv", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExportTransactionsCSV([]Transaction{{
			Date: "2026-05-08", ID: "tx_1", AccountID: "acc_1", Merchant: "Store", PlaidName: "STORE LLC",
			Category: "Food", CategoryGroup: TransactionCategoryGroup{Name: "Groceries", Type: "expense"},
			Amount: -20, Notes: "lunch", Tags: []Tag{{ID: "t1", Name: "work"}}, IsRecurring: true,
			Splits: []TransactionSplit{{ID: "sp_1", Amount: -20, Category: "Food", Merchant: "Store", Notes: "lunch"}},
		}}, &buf)
		if err != nil {
			t.Fatalf("ExportTransactionsCSV() error = %v", err)
		}
		for _, want := range []string{
			"Date,ID,AccountID,Merchant,PlaidName,ProviderDescription,Category,CategoryGroup,CategoryGroupType,Amount,Notes,Tags,Goal,Pending,HideFromReports,IsRecurring,ReviewStatus,NeedsReview,Splits",
			"2026-05-08,tx_1,acc_1,Store,STORE LLC,,Food,Groceries,expense,-20.00,lunch,work,,false,false,true,,false,Store|Food|-20.00|lunch",
		} {
			if !strings.Contains(buf.String(), want) {
				t.Fatalf("ExportTransactionsCSV() output = %q", buf.String())
			}
		}
	})

	t.Run("export csv header error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{failOnCall: 1} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})

	t.Run("export csv row error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{failOnCall: 2} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})

	t.Run("export csv flush error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{err: errors.New("flush failed")} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})
}

func TestServiceErrorBranches(t *testing.T) {
	t.Run("accounts and reference methods", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetAccounts", nil, func(s *Service) error { _, err := s.ListAccounts(context.Background()); return err })
		runGraphQLErrorCase(t, "AccountDetails_getAccount", map[string]any{"id": "acc-1"}, func(s *Service) error { _, err := s.GetAccount(context.Background(), "acc-1"); return err })
		runGraphQLErrorCase(t, "GetAccountTypeOptions", nil, func(s *Service) error { _, err := s.GetAccountTypes(context.Background()); return err })
		runGraphQLErrorCase(t, "ForceRefreshAccountsQuery", nil, func(s *Service) error { _, err := s.GetAccountsRefreshStatus(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_CreateManualAccount", map[string]any{"input": map[string]any{"type": "bank", "subtype": "checking", "includeInNetWorth": true, "name": "Savings", "displayBalance": 10.0}}, func(s *Service) error {
			_, err := s.CreateManualAccount(context.Background(), "Savings", "bank", "checking", 10)
			return err
		})
		runGraphQLErrorCase(t, "Common_UpdateAccount", map[string]any{"input": map[string]any{"id": "acc-1"}}, func(s *Service) error { _, err := s.UpdateAccount(context.Background(), "acc-1", nil, nil); return err })
		runGraphQLErrorCase(t, "Common_ForceRefreshAccountsMutation", map[string]any{"input": map[string]any{}}, func(s *Service) error { return s.RefreshAccounts(context.Background(), nil) })
		runGraphQLErrorCase(t, "Common_DeleteAccount", map[string]any{"id": "acc-1"}, func(s *Service) error { return s.DeleteAccount(context.Background(), "acc-1") })
		runGraphQLErrorCase(t, "Web_GetHoldings", nil, func(s *Service) error { _, err := s.GetAccountHoldings(context.Background(), "acc-1"); return err })
		runGraphQLErrorCase(t, "GetAccountHistory", map[string]any{"filters": map[string]any{}}, func(s *Service) error {
			_, err := s.GetAccountHistory(context.Background(), "acc-1", "", "")
			return err
		})
		runGraphQLErrorCase(t, "GetAccountRecentBalances", map[string]any{"startDate": "2026-05-01"}, func(s *Service) error {
			_, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01")
			return err
		})
		runGraphQLErrorCase(t, "Common_GetDisplayBalanceAtDate", map[string]any{"date": "2026-05-10"}, func(s *Service) error {
			_, err := s.GetAccountBalancesAt(context.Background(), "2026-05-10", nil)
			return err
		})
		runGraphQLErrorCase(t, "GetSnapshotsByAccountType", map[string]any{"startDate": "2026-05-01", "timeframe": "month"}, func(s *Service) error {
			_, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month")
			return err
		})
		runGraphQLErrorCase(t, "GetAggregateSnapshots", map[string]any{"filters": map[string]any{"startDate": "2026-05-01"}}, func(s *Service) error {
			_, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "", "")
			return err
		})
	})

	t.Run("budgets cashflow and lookup", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_GetJointPlanningData", map[string]any{"startDate": "2026-05-01", "endDate": "2026-05-31"}, func(s *Service) error {
			_, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "Common_GetJointPlanningData", map[string]any{"startDate": "", "endDate": ""}, func(s *Service) error {
			_, err := s.ListBudgets(context.Background(), ListBudgetsOptions{})
			return err
		})
		runGraphQLErrorCase(t, "Common_UpdateBudgetItem", map[string]any{"input": map[string]any{"categoryId": "cat-1", "amount": 10.0, "timeframe": "month", "startDate": "2026-05-01", "applyToFuture": false}}, func(s *Service) error {
			_, err := s.SetBudget(context.Background(), "cat-1", 10, "2026-05-01")
			return err
		})
		runGraphQLErrorCase(t, "Common_ResetBudget", map[string]any{"input": map[string]any{"startDate": "2026-01-01", "overwriteExisting": false}}, func(s *Service) error { return s.ResetBudget(context.Background(), "2026-01-01", false, nil) })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]any{"offset": 0, "limit": 1000, "filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.ListCashflow(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowSummary", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowCategories", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowMerchants", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetAggregatesGraph", map[string]any{"filters": map[string]any{"startDate": "2026-01-01", "endDate": "2026-03-31"}}, func(s *Service) error {
			_, err := s.GetCashflowTrends(context.Background(), &CashflowTrendOptions{StartDate: "2026-01-01", EndDate: "2026-03-31", GroupBy: "category", Period: "month"})
			return err
		})
		runGraphQLErrorCase(t, "GetTransactionRules", nil, func(s *Service) error { _, err := s.ListRules(context.Background()); return err })
		runGraphQLErrorCase(t, "ManageGetCategoryGroups", nil, func(s *Service) error { _, err := s.ListCategoryGroups(context.Background()); return err })
		runGraphQLErrorCase(t, "GetCategories", nil, func(s *Service) error { _, err := s.ListCategories(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_CreateCategory", map[string]any{"input": map[string]any{"name": "Food", "group": "g1"}}, func(s *Service) error { _, err := s.CreateCategory(context.Background(), "Food", "g1", ""); return err })
		runGraphQLErrorCase(t, "GetCreditScoreSnapshots", nil, func(s *Service) error { _, err := s.GetCreditHistory(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_GetInstitutionSettings", nil, func(s *Service) error { _, err := s.ListInstitutions(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]any{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]any{}}, func(s *Service) error {
			_, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01")
			return err
		})
		runGraphQLErrorCase(t, "UpdateRecurringTransaction", map[string]any{"id": "r1", "amount": 21.5}, func(s *Service) error { _, err := s.UpdateRecurring(context.Background(), "r1", 21.5); return err })
		runGraphQLErrorCase(t, "GetSubscriptionDetails", nil, func(s *Service) error { _, err := s.GetSubscriptionDetails(context.Background()); return err })
		runGraphQLErrorCase(t, "Common_SavingsGoals", nil, func(s *Service) error { _, err := s.ListGoals(context.Background()); return err })
		runGraphQLErrorCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, func(s *Service) error { _, err := s.ListTags(context.Background(), "", 0); return err })
		runGraphQLErrorCase(t, "Common_CreateTransactionTag", map[string]any{"input": map[string]any{"name": "Trip", "color": "blue"}}, func(s *Service) error { _, err := s.CreateTag(context.Background(), "Trip", "blue"); return err })
	})

	t.Run("transactions", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetTransactionDrawer", map[string]any{"id": "tx-1"}, func(s *Service) error { _, err := s.GetTransaction(context.Background(), "tx-1"); return err })
		runGraphQLErrorCase(t, "GetTransactionsPage", map[string]any{"filters": map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error {
			_, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]any{"limit": 1000, "offset": 0, "filters": map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error {
			_, err := s.GetDuplicateTransactions(context.Background(), "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]any{"input": map[string]any{"id": "tx-1"}}, func(s *Service) error {
			_, err := s.UpdateTransaction(context.Background(), "tx-1", nil, nil, nil, nil, nil, nil, nil)
			return err
		})
		runGraphQLErrorCase(t, "Common_DeleteTransactionMutation", map[string]any{"input": map[string]any{"transactionId": "tx-1"}}, func(s *Service) error { return s.DeleteTransaction(context.Background(), "tx-1") })
		runGraphQLErrorCase(t, "Common_CreateTransactionMutation", map[string]any{"input": map[string]any{"date": "2026-05-08", "accountId": "", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "", "shouldUpdateBalance": false}}, func(s *Service) error {
			_, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "", "")
			return err
		})
		runGraphQLErrorCase(t, "Web_SetTransactionTags", map[string]any{"input": map[string]any{"transactionId": "tx-1", "tagIds": []string{"tag-1"}}}, func(s *Service) error { return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1"}) })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]any{"limit": 10, "offset": 0, "filters": map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, _, err := s.ListTransactions(context.Background(), &ListTransactionsOptions{Limit: 10})
			return err
		})
	})

	t.Run("investments", func(t *testing.T) {
		runGraphQLErrorCase(t, "Web_GetPortfolio", map[string]any{"portfolioInput": map[string]any{}}, func(s *Service) error {
			_, err := s.GetInvestmentPortfolio(context.Background(), InvestmentPortfolioOptions{})
			return err
		})
		runGraphQLErrorCase(t, "Web_GetSecuritiesHistoricalPerformance", map[string]any{"input": map[string]any{"securityIds": []string{"sec-1"}, "startDate": "2026-01-01", "endDate": "2026-05-10"}}, func(s *Service) error {
			_, err := s.GetSecurityPerformance(context.Background(), SecurityPerformanceOptions{SecurityIDs: []string{"sec-1"}, StartDate: "2026-01-01", EndDate: "2026-05-10"})
			return err
		})
	})
}

func TestServiceHTTPHelpers(t *testing.T) {
	testServiceHTTPDownloadAttachmentPaths(t)
	testServiceHTTPUploadBalanceHistoryPaths(t)
	testServiceHTTPAttachmentAvailabilityPaths(t)
	testServiceReceiptUploadPaths(t)
}

func testServiceHTTPDownloadAttachmentPaths(t *testing.T) {
	t.Helper()

	t.Run("download attachment success", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			mustEq(t, "GET", req.Method)
			mustEq(t, "https://files.example/attachment.csv", req.URL.String())
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("hello"))}, nil
		})

		var buf bytes.Buffer
		svc := newMockService("token-123")
		mustNoErr(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
		eq(t, "hello", buf.String())
	})

	t.Run("download attachment error", func(t *testing.T) {
		svc := newMockService("token-123")
		hasErr(t, svc.DownloadAttachment(context.Background(), "://", io.Discard))
	})

	t.Run("download attachment transport error", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		var buf bytes.Buffer
		svc := newMockService("token-123")
		hasErr(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
	})

	t.Run("download attachment non-200", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		var buf bytes.Buffer
		svc := newMockService("token-123")
		hasErr(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
	})
}

func testServiceHTTPUploadBalanceHistoryPaths(t *testing.T) {
	t.Helper()

	newCSV := func(t *testing.T) io.Reader {
		t.Helper()
		return strings.NewReader("Date,Amount,Account Name\n2026-01-01,100,Checking\n")
	}

	t.Run("upload account balance history completed on parse", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			mustEq(t, "POST", req.Method)
			mustEq(t, "https://api.monarch.com/account-balance-history/upload/", req.URL.String())
			mustEq(t, "web", req.Header.Get("Client-Platform"))
			mustEq(t, "Token tok", req.Header.Get("Authorization"))
			body, err := io.ReadAll(req.Body)
			mustNoErr(t, err)
			hasSubstr(t, string(body), `name="files"`)
			hasSubstr(t, string(body), "account_files_mapping")
			hasSubstr(t, string(body), `"upload.csv":"acc-1"`)
			hasSubstr(t, string(body), "2026-01-01,100,Checking")
			return testutil.JSONResponse(`{"session_key":"sk-1"}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Web_ParseUploadBalanceHistorySession")
				expectVars(t, req.Variables, map[string]any{"input": map[string]any{"sessionKey": "sk-1"}})
				return client.respond(result, `{"parseBalanceHistory":{"uploadBalanceHistorySession":{"sessionKey":"sk-1","status":"completed"}}}`)
			},
		}

		mustNoErr(t, NewService(client).UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})

	t.Run("upload account balance history polls until completed", func(t *testing.T) {
		origDelay := balanceHistoryPollDelay
		balanceHistoryPollDelay = time.Millisecond
		defer func() { balanceHistoryPollDelay = origDelay }()

		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{"session_key":"sk-2"}`), nil
		})

		var polls int
		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Web_ParseUploadBalanceHistorySession":
					return client.respond(result, `{"parseBalanceHistory":{"uploadBalanceHistorySession":{"status":"processing"}}}`)
				case "Web_GetUploadBalanceHistorySession":
					polls++
					status := "processing"
					if polls >= 2 {
						status = "completed"
					}
					return client.respond(result, fmt.Sprintf(`{"uploadBalanceHistorySession":{"sessionKey":"sk-2","status":%q}}`, status))
				default:
					t.Fatalf("unexpected operation %s", req.OperationName)
					return nil
				}
			},
		}

		mustNoErr(t, NewService(client).UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
		eq(t, 2, polls)
	})

	t.Run("upload account balance history poll timeout", func(t *testing.T) {
		origDelay := balanceHistoryPollDelay
		origTimeout := balanceHistoryPollTimeout
		balanceHistoryPollDelay = time.Millisecond
		balanceHistoryPollTimeout = 5 * time.Millisecond
		defer func() {
			balanceHistoryPollDelay = origDelay
			balanceHistoryPollTimeout = origTimeout
		}()

		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{"session_key":"sk-3"}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Web_ParseUploadBalanceHistorySession":
					return client.respond(result, `{"parseBalanceHistory":{"uploadBalanceHistorySession":{"status":"processing"}}}`)
				case "Web_GetUploadBalanceHistorySession":
					return client.respond(result, `{"uploadBalanceHistorySession":{"status":"processing"}}`)
				default:
					t.Fatalf("unexpected operation %s", req.OperationName)
					return nil
				}
			},
		}

		hasErr(t, NewService(client).UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})

	t.Run("upload account balance history non-200", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		svc := NewService(&mockClient{token: "tok"})
		hasErr(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})

	t.Run("upload account balance history network error", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		svc := NewService(&mockClient{token: "tok"})
		hasErr(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})

	t.Run("upload account balance history missing session key", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{"ok":true}`), nil
		})

		svc := NewService(&mockClient{token: "tok"})
		err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t))
		errContains(t, err, "session_key")
	})

	t.Run("upload account balance history request error", func(t *testing.T) {
		original := newBalanceHistoryRequest
		newBalanceHistoryRequest = func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("request failed")
		}
		defer func() { newBalanceHistoryRequest = original }()

		svc := NewService(&mockClient{token: "tok"})
		hasErr(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})

	t.Run("upload account balance history read error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "tok"})
		hasErr(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", testutil.FailingReader{}))
	})

	t.Run("upload account balance history form file error", func(t *testing.T) {
		original := createBalanceHistoryFormFile
		createBalanceHistoryFormFile = func(*multipart.Writer, string, string) (io.Writer, error) {
			return nil, errors.New("form file failed")
		}
		defer func() { createBalanceHistoryFormFile = original }()

		svc := NewService(&mockClient{token: "tok"})
		hasErr(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", newCSV(t)))
	})
}

func testServiceHTTPAttachmentAvailabilityPaths(t *testing.T) {
	t.Helper()

	t.Run("list transaction attachments", func(t *testing.T) {
		got, err := NewService(&mockClient{token: "tok"}).ListTransactionAttachments(context.Background(), "tx-1")
		mustNoErr(t, err)
		isEmpty(t, got)
	})

	t.Run("upload attachment", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			mustEq(t, "POST", req.Method)
			mustEq(t, "https://api.cloudinary.com/v1_1/monarch-money/image/upload/", req.URL.String())
			body, err := io.ReadAll(req.Body)
			mustNoErr(t, err)
			hasSubstr(t, string(body), `filename="receipt.pdf"`)
			hasSubstr(t, string(body), `name="api_key"`)
			hasSubstr(t, string(body), "key-1")
			hasSubstr(t, string(body), `name="upload_preset"`)
			return testutil.JSONResponse(`{"public_id":"pub-1","format":"pdf","bytes":3}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Common_GetTransactionAttachmentUploadInfo":
					assertReq(t, req, "Common_GetTransactionAttachmentUploadInfo")
					expectVars(t, req.Variables, map[string]any{"transactionId": "tx-1"})
					return client.respond(result, `{"getTransactionAttachmentUploadInfo":{"info":{"requestParams":{"timestamp":1700000000,"folder":"monarch","signature":"sig-1","api_key":"key-1","upload_preset":"preset-1"}}}}`)
				case "Common_AddTransactionAttachment":
					expectVars(t, req.Variables, map[string]any{"input": map[string]any{
						"extension":     "pdf",
						"transactionId": "tx-1",
						"filename":      "receipt.pdf",
						"publicId":      "pub-1",
						"sizeBytes":     3,
					}})
					return client.respond(result, `{"addTransactionAttachment":{"errors":null}}`)
				default:
					t.Fatalf("unexpected operation %s", req.OperationName)
					return nil
				}
			},
		}

		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		mustNoErr(t, os.WriteFile(tmp, []byte("pdf"), 0o600))
		mustNoErr(t, NewService(client).UploadAttachment(context.Background(), "tx-1", tmp))
	})

	t.Run("upload attachment upload info error", func(t *testing.T) {
		client := &mockClient{token: "tok", handler: func(*graphql.Request, any) error { return errors.New("boom") }}
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		mustNoErr(t, os.WriteFile(tmp, []byte("pdf"), 0o600))
		hasErr(t, NewService(client).UploadAttachment(context.Background(), "tx-1", tmp))
	})

	t.Run("upload attachment cloudinary non-200", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"getTransactionAttachmentUploadInfo":{"info":{"requestParams":{"timestamp":1700000000,"folder":"monarch","signature":"sig-1","api_key":"key-1","upload_preset":"preset-1"}}}}`)
			},
		}
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		mustNoErr(t, os.WriteFile(tmp, []byte("pdf"), 0o600))
		hasErr(t, NewService(client).UploadAttachment(context.Background(), "tx-1", tmp))
	})

	t.Run("upload attachment missing public id", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{"bytes":3}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"getTransactionAttachmentUploadInfo":{"info":{"requestParams":{"timestamp":1700000000,"folder":"monarch","signature":"sig-1","api_key":"key-1","upload_preset":"preset-1"}}}}`)
			},
		}
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		mustNoErr(t, os.WriteFile(tmp, []byte("pdf"), 0o600))
		err := NewService(client).UploadAttachment(context.Background(), "tx-1", tmp)
		errContains(t, err, "public_id")
	})

	t.Run("upload attachment add errors", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{"public_id":"pub-1","format":"pdf","bytes":3}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Common_GetTransactionAttachmentUploadInfo":
					return client.respond(result, `{"getTransactionAttachmentUploadInfo":{"info":{"requestParams":{"timestamp":1700000000,"folder":"monarch","signature":"sig-1","api_key":"key-1","upload_preset":"preset-1"}}}}`)
				default:
					return client.respond(result, `{"addTransactionAttachment":{"errors":{"message":"denied"}}}`)
				}
			},
		}
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		mustNoErr(t, os.WriteFile(tmp, []byte("pdf"), 0o600))
		err := NewService(client).UploadAttachment(context.Background(), "tx-1", tmp)
		errContains(t, err, "denied")
	})

	t.Run("upload attachment file read error", func(t *testing.T) {
		hasErr(t, NewService(&mockClient{token: "tok"}).UploadAttachment(context.Background(), "tx-1", filepath.Join(t.TempDir(), "missing.pdf")))
	})
}

func testServiceReceiptUploadPaths(t *testing.T) {
	t.Helper()

	newReceipt := func(t *testing.T) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "receipt.jpg")
		mustNoErr(t, os.WriteFile(path, []byte("jpg"), 0o600))
		return path
	}

	t.Run("upload receipt to inbox", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			mustEq(t, "POST", req.Method)
			mustEq(t, "https://api.monarch.com/retail-sync/sync-1/files", req.URL.String())
			mustEq(t, "Token tok", req.Header.Get("Authorization"))
			body, err := io.ReadAll(req.Body)
			mustNoErr(t, err)
			hasSubstr(t, string(body), `name="payloads_count"`)
			hasSubstr(t, string(body), `"vendor":"user_import"`)
			hasSubstr(t, string(body), `filename="receipt.jpg"`)
			return testutil.JSONResponse(`{}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Common_CreateBulkRetailSync":
					expectVars(t, req.Variables, map[string]any{"input": map[string]any{"count": 1}})
					return client.respond(result, `{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1","vendor":"user_import","status":"created"}],"errors":null}}`)
				case "Common_StartRetailSync":
					assertReq(t, req, "Common_StartRetailSync")
					expectVars(t, req.Variables, map[string]any{"syncId": "sync-1"})
					return client.respond(result, `{"startRetailSync":{"retailSync":{"id":"sync-1","vendor":"user_import","status":"started","startedAt":"2026-05-01T00:00:00Z"},"errors":null}}`)
				default:
					t.Fatalf("unexpected operation %s", req.OperationName)
					return nil
				}
			},
		}

		syncResult, err := NewService(client).UploadReceiptToInbox(context.Background(), newReceipt(t))
		mustNoErr(t, err)
		eq(t, "sync-1", syncResult.ID)
		eq(t, "started", syncResult.Status)
		eq(t, "2026-05-01T00:00:00Z", syncResult.StartedAt)
	})

	t.Run("upload receipt create session errors", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"createBulkRetailSync":{"retailSyncs":[],"errors":{"message":"no quota"}}}`)
			},
		}
		_, err := NewService(client).UploadReceiptToInbox(context.Background(), newReceipt(t))
		errContains(t, err, "no quota")
	})

	t.Run("upload receipt create session empty syncs", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"createBulkRetailSync":{"retailSyncs":[],"errors":null}}`)
			},
		}
		_, err := NewService(client).UploadReceiptToInbox(context.Background(), newReceipt(t))
		errContains(t, err, "failed to create retail sync session")
	})

	t.Run("upload receipt file non-200", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 413, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`)
			},
		}
		_, err := NewService(client).UploadReceiptToInbox(context.Background(), newReceipt(t))
		errContains(t, err, "status 413")
	})

	t.Run("upload receipt start errors", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = testutil.RoundTripFunc(func(*http.Request) (*http.Response, error) {
			return testutil.JSONResponse(`{}`), nil
		})

		var client *mockClient
		client = &mockClient{
			token: "tok",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Common_CreateBulkRetailSync":
					return client.respond(result, `{"createBulkRetailSync":{"retailSyncs":[{"id":"sync-1"}],"errors":null}}`)
				default:
					return client.respond(result, `{"startRetailSync":{"retailSync":null,"errors":{"message":"expired"}}}`)
				}
			},
		}
		_, err := NewService(client).UploadReceiptToInbox(context.Background(), newReceipt(t))
		errContains(t, err, "expired")
	})

	t.Run("upload receipt read error", func(t *testing.T) {
		_, err := NewService(&mockClient{token: "tok"}).UploadReceiptToInbox(context.Background(), filepath.Join(t.TempDir(), "missing.jpg"))
		hasErr(t, err)
	})
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func hasErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func errContains(t *testing.T, err error, sub string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), sub) {
		t.Fatalf("error = %v, want containing %q", err, sub)
	}
}

func eq(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func mustEq(t *testing.T, want, got any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func mustLen(t *testing.T, v any, n int) {
	t.Helper()
	if l := reflect.ValueOf(v).Len(); l != n {
		t.Fatalf("len = %d, want %d", l, n)
	}
}

func mustNotNil(t *testing.T, v any) {
	t.Helper()
	if isNilValue(v) {
		t.Fatal("got nil, want non-nil")
	}
}

func isTrue(t *testing.T, cond bool) {
	t.Helper()
	if !cond {
		t.Error("got false, want true")
	}
}

func hasSubstr(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("%q does not contain %q", s, sub)
	}
}

func isEmpty(t *testing.T, v any) {
	t.Helper()
	if l := reflect.ValueOf(v).Len(); l != 0 {
		t.Errorf("len = %d, want 0", l)
	}
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

func TestGetFinancialOverview(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	var client *mockClient
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result any) error {
			mu.Lock()
			calls++
			mu.Unlock()
			switch req.OperationName {
			case "GetAccounts":
				return client.respond(result, `{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank"},"displayBalance":1000,"isAsset":true,"includeInNetWorth":true}]}`)
			case "GetCashflowSummary":
				return client.respond(result, `{"aggregates":[{"summary":{"sumIncome":200,"sumExpense":100,"savings":100,"savingsRate":50}}]}`)
			case "GetTransactionsList":
				return client.respond(result, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-01-01","amount":10,"merchant":{"name":"Paycheck"},"category":{"name":"Income"}}],"totalCount":1}}`)
			default:
				return fmt.Errorf("unexpected operation %s", req.OperationName)
			}
		},
	}

	ov, err := NewService(client).GetFinancialOverview(context.Background(), "2026-01-01", "2026-01-31")
	mustNoErr(t, err)
	mustNotNil(t, ov)
	eq(t, 1, ov.AccountCount)
	eq(t, 1000.0, ov.NetWorth)
	mustNotNil(t, ov.Cashflow)
	eq(t, 50.0, ov.Cashflow.SavingsRate)
	eq(t, 1, ov.TransactionTotal)
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (concurrent fetches)", calls)
	}
}

func TestGetFinancialOverviewDefaultsToCurrentMonth(t *testing.T) {
	var client *mockClient
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result any) error {
			switch req.OperationName {
			case "GetAccounts":
				return client.respond(result, `{"accounts":[]}`)
			case "GetCashflowSummary":
				return client.respond(result, `{"aggregates":[{"summary":{"sumIncome":0,"sumExpense":0,"savings":0,"savingsRate":0}}]}`)
			case "GetTransactionsList":
				return client.respond(result, `{"allTransactions":{"results":[],"totalCount":0}}`)
			default:
				return fmt.Errorf("unexpected operation %s", req.OperationName)
			}
		},
	}

	ov, err := NewService(client).GetFinancialOverview(context.Background(), "", "")
	mustNoErr(t, err)
	mustNotNil(t, ov)
	eq(t, 0, ov.AccountCount)
}

func TestGetFinancialOverviewPropagatesErrors(t *testing.T) {
	client := &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result any) error {
			return errors.New("boom")
		},
	}

	_, err := NewService(client).GetFinancialOverview(context.Background(), "2026-01-01", "2026-01-31")
	hasErr(t, err)
}

func TestServiceTagUpdatePaths(t *testing.T) {
	t.Run("update tag name only keeps color", func(t *testing.T) {
		var mu sync.Mutex
		calls := 0
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				mu.Lock()
				calls++
				mu.Unlock()
				if req.OperationName == "Common_GetHouseholdTransactionTags" {
					return client.respond(result, `{"householdTransactionTags":[{"id":"tag-1","name":"Old","color":"blue","order":1}]}`)
				}
				if req.OperationName == "Common_UpdateTransactionTag" {
					input, _ := req.Variables["input"].(map[string]any)
					if input["id"] != "tag-1" || input["name"] != "New" || input["color"] != "blue" {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"updateTransactionTag":{"tag":{"id":"tag-1","name":"New","color":"blue","order":1},"errors":null}}`)
				}
				t.Fatalf("operation = %q", req.OperationName)
				return nil
			},
		}
		got, err := NewService(client).UpdateTag(context.Background(), "tag-1", strPtr("New"), nil)
		mustNoErr(t, err)
		eq(t, "New", got.Name)
		eq(t, "blue", got.Color)
	})

	t.Run("update tag missing", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"householdTransactionTags":[]}`)
			},
		}
		_, err := NewService(client).UpdateTag(context.Background(), "nope", strPtr("x"), nil)
		hasErr(t, err)
	})

	t.Run("update tag mutation error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_UpdateTransactionTag" {
					return client.respond(result, `{"updateTransactionTag":{"tag":null,"errors":[{"message":"taken"}]}}`)
				}
				return client.respond(result, `{"householdTransactionTags":[{"id":"tag-1","name":"Old","color":"blue"}]}`)
			},
		}
		_, err := NewService(client).UpdateTag(context.Background(), "tag-1", strPtr("x"), nil)
		hasErr(t, err)
	})

	t.Run("update tag missing tag payload", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_UpdateTransactionTag" {
					return client.respond(result, `{"updateTransactionTag":{"tag":null,"errors":null}}`)
				}
				return client.respond(result, `{"householdTransactionTags":[{"id":"tag-1","name":"Old","color":"blue"}]}`)
			},
		}
		_, err := NewService(client).UpdateTag(context.Background(), "tag-1", strPtr("x"), nil)
		hasErr(t, err)
	})

	t.Run("delete tag error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_DeleteHouseholdTransactionTag", map[string]any{"tagId": "tag-1"}, func(s *Service) error {
			return s.DeleteTag(context.Background(), "tag-1")
		})
	})

	t.Run("delete tag payload error", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHouseholdTransactionTag", map[string]any{"tagId": "tag-1"}, `{"deleteTransactionTag":{"errors":[{"message":"used"}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteTag(context.Background(), "tag-1"))
			return nil
		})
	})

	t.Run("tag without order sorts last", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, `{"householdTransactionTags":[{"id":"tag-2","name":"b","color":"red"},{"id":"tag-1","name":"a","color":"blue","order":1}]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background(), "", 0)
			mustNoErr(t, err)
			mustLen(t, got, 2)
			eq(t, "tag-1", got[0].ID)
			eq(t, "tag-2", got[1].ID)
			return nil
		})
	})

	t.Run("list tags with search and limit", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{"search": "vac", "limit": 5}, `{"householdTransactionTags":[]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background(), "vac", 5)
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("create tag error", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionTag", map[string]any{"input": map[string]any{"name": "x", "color": "blue"}}, `{"createTransactionTag":{"tag":null,"errors":[{"message":"taken"}]}}`, func(s *Service) error {
			_, err := s.CreateTag(context.Background(), "x", "blue")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create tag missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionTag", map[string]any{"input": map[string]any{"name": "x", "color": "blue"}}, `{"createTransactionTag":{"tag":null,"errors":null}}`, func(s *Service) error {
			_, err := s.CreateTag(context.Background(), "x", "blue")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("reorder tag error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_UpdateTransactionTagOrder", map[string]any{"tagId": "tag-1", "order": 2}, func(s *Service) error {
			_, err := s.ReorderTag(context.Background(), "tag-1", 2)
			return err
		})
	})
}

func strPtr(s string) *string { return &s }

func TestServiceTagSortTies(t *testing.T) {
	t.Run("name and id tiebreaks", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdTransactionTags", map[string]any{}, `{"householdTransactionTags":[{"id":"tag-b","name":"same","color":"red","order":1},{"id":"tag-a","name":"same","color":"blue","order":1},{"id":"tag-c","name":"aaa","color":"green","order":1}]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background(), "", 0)
			mustNoErr(t, err)
			mustLen(t, got, 3)
			eq(t, "tag-c", got[0].ID)
			eq(t, "tag-a", got[1].ID)
			eq(t, "tag-b", got[2].ID)
			return nil
		})
	})

	t.Run("delete tag fallback error", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHouseholdTransactionTag", map[string]any{"tagId": "tag-1"}, `{"deleteTransactionTag":{"errors":[{}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteTag(context.Background(), "tag-1"))
			return nil
		})
	})
}

func TestServiceMerchantPaths(t *testing.T) {
	t.Run("list merchants with filters", func(t *testing.T) {
		runGraphQLCase(t, "Common_ListMerchants", map[string]any{"search": "Whole", "limit": 10, "offset": 5, "orderBy": "name"}, `{"merchants":[{"id":"m-1","name":"Whole Foods","logoUrl":"","transactionCount":42,"createdAt":"2026-01-01","recurringTransactionStream":null}]}`, func(s *Service) error {
			got, err := s.ListMerchants(context.Background(), &ListMerchantsOptions{Search: "Whole", Limit: 10, Offset: 5, OrderBy: "name"})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Whole Foods", got[0].Name)
			eq(t, 42, got[0].TransactionCount)
			return nil
		})
	})

	t.Run("list merchants bare", func(t *testing.T) {
		runGraphQLCase(t, "Common_ListMerchants", map[string]any{}, `{"merchants":[{"id":"m-1","name":"Whole Foods","transactionCount":1,"recurringTransactionStream":{"id":"rs-1"}}]}`, func(s *Service) error {
			got, err := s.ListMerchants(context.Background(), nil)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "rs-1", got[0].RecurringStreamID)
			return nil
		})
	})

	t.Run("list merchants with default-category filter", func(t *testing.T) {
		hasDefault := true
		includeAll := true
		runGraphQLCase(t, "Common_ListMerchants", map[string]any{"search": "Whole", "filters": map[string]any{"hasDefaultCategory": true}, "includeIds": []string{"m-1"}, "includeMerchantsWithoutTransactions": true}, `{"merchants":[{"id":"m-1","name":"Whole Foods","transactionCount":42,"defaultCategory":{"id":"cat-1","name":"Groceries"},"defaultCategoryApplicationMode":"new_and_edits","showDefaultCategoryPrompt":false,"recurringTransactionStream":null}]}`, func(s *Service) error {
			got, err := s.ListMerchants(context.Background(), &ListMerchantsOptions{Search: "Whole", HasDefaultCategory: &hasDefault, IncludeIDs: []string{"m-1"}, IncludeMerchantsWithoutTransactions: &includeAll})
			mustNoErr(t, err)
			mustLen(t, got, 1)
			mustNotNil(t, got[0].DefaultCategory)
			eq(t, "Groceries", got[0].DefaultCategory.Name)
			eq(t, "new_and_edits", got[0].DefaultCategoryApplicationMode)
			mustNotNil(t, got[0].ShowDefaultCategoryPrompt)
			return nil
		})
	})

	t.Run("get merchant", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetEditMerchant", map[string]any{"merchantId": "m-1"}, `{"merchant":{"id":"m-1","name":"Whole Foods","transactionCount":42,"ruleCount":3,"canBeDeleted":true,"createdAt":"2026-01-01"}}`, func(s *Service) error {
			got, err := s.GetMerchant(context.Background(), "m-1")
			mustNoErr(t, err)
			eq(t, "Whole Foods", got.Name)
			return nil
		})
	})

	t.Run("get merchant missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetEditMerchant", map[string]any{"merchantId": "nope"}, `{"merchant":null}`, func(s *Service) error {
			_, err := s.GetMerchant(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update merchant", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMerchant", map[string]any{"input": map[string]any{"merchantId": "m-1", "name": "WF"}}, `{"updateMerchant":{"merchant":{"id":"m-1","name":"WF"},"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateMerchant(context.Background(), &UpdateMerchantInput{ID: "m-1", Name: "WF"})
			mustNoErr(t, err)
			eq(t, "WF", got.Name)
			return nil
		})
	})

	t.Run("update merchant error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMerchant", map[string]any{"input": map[string]any{"merchantId": "m-1", "name": "WF"}}, `{"updateMerchant":{"merchant":null,"errors":[{"message":"taken"}]}}`, func(s *Service) error {
			_, err := s.UpdateMerchant(context.Background(), &UpdateMerchantInput{ID: "m-1", Name: "WF"})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update merchant missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMerchant", map[string]any{"input": map[string]any{"merchantId": "m-1", "name": "WF"}}, `{"updateMerchant":{"merchant":null,"errors":null}}`, func(s *Service) error {
			_, err := s.UpdateMerchant(context.Background(), &UpdateMerchantInput{ID: "m-1", Name: "WF"})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update merchant default category", func(t *testing.T) {
		prompt := true
		runGraphQLCase(t, "Common_UpdateMerchant", map[string]any{"input": map[string]any{"merchantId": "m-1", "defaultCategoryId": "cat-1", "defaultCategoryApplicationMode": "new_only", "showDefaultCategoryPrompt": true, "recurrence": map[string]any{"isRecurring": true}}}, `{"updateMerchant":{"merchant":{"id":"m-1","name":"Whole Foods","defaultCategory":{"id":"cat-1","name":"Groceries"},"defaultCategoryApplicationMode":"new_only","showDefaultCategoryPrompt":true},"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateMerchant(context.Background(), &UpdateMerchantInput{ID: "m-1", DefaultCategoryID: "cat-1", DefaultCategoryMode: "new_only", ShowDefaultCategoryPrompt: &prompt, Recurrence: map[string]any{"isRecurring": true}})
			mustNoErr(t, err)
			mustNotNil(t, got.DefaultCategory)
			eq(t, "cat-1", got.DefaultCategory.ID)
			eq(t, "new_only", got.DefaultCategoryApplicationMode)
			return nil
		})
	})

	t.Run("delete merchant", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteMerchant", map[string]any{"merchantId": "m-1"}, `{"deleteMerchant":{"success":true}}`, func(s *Service) error {
			return s.DeleteMerchant(context.Background(), "m-1", "")
		})
	})

	t.Run("delete merchant with move target", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteMerchant", map[string]any{"merchantId": "m-1", "moveToId": "m-2"}, `{"deleteMerchant":{"success":true}}`, func(s *Service) error {
			return s.DeleteMerchant(context.Background(), "m-1", "m-2")
		})
	})

	t.Run("delete merchant failure", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteMerchant", map[string]any{"merchantId": "m-1"}, `{"deleteMerchant":{"success":false}}`, func(s *Service) error {
			hasErr(t, s.DeleteMerchant(context.Background(), "m-1", ""))
			return nil
		})
	})

	t.Run("merchant error paths", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_ListMerchants", map[string]any{}, func(s *Service) error {
			_, err := s.ListMerchants(context.Background(), nil)
			return err
		})
		runGraphQLErrorCase(t, "Common_GetEditMerchant", map[string]any{"merchantId": "m-1"}, func(s *Service) error {
			_, err := s.GetMerchant(context.Background(), "m-1")
			return err
		})
	})
}

func TestServiceHouseholdPaths(t *testing.T) {
	t.Run("get household", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetMyHousehold", nil, `{"myHousehold":{"id":"hh-1","name":"Smiths","address":"1 Main St","city":"SF","state":"CA","zipCode":"94101","country":"US"}}`, func(s *Service) error {
			got, err := s.GetHousehold(context.Background())
			mustNoErr(t, err)
			eq(t, "Smiths", got.Name)
			eq(t, "94101", got.ZipCode)
			return nil
		})
	})

	t.Run("get household missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetMyHousehold", nil, `{"myHousehold":null}`, func(s *Service) error {
			_, err := s.GetHousehold(context.Background())
			hasErr(t, err)
			return nil
		})
	})

	t.Run("list household members", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdMembers", nil, `{"myHousehold":{"id":"hh-1","users":[{"id":"u-1","name":"Ann","displayName":"Ann S","email":"ann@example.com","householdRole":"owner","hasMfaOn":true}]}}`, func(s *Service) error {
			got, err := s.ListHouseholdMembers(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "ann@example.com", got[0].Email)
			return nil
		})
	})

	t.Run("list household members nil household", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdMembers", nil, `{"myHousehold":null}`, func(s *Service) error {
			got, err := s.ListHouseholdMembers(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("get household member", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdMembers", nil, `{"myHousehold":{"id":"hh-1","users":[{"id":"u-1","name":"Ann","displayName":"Ann S","email":"ann@example.com","householdRole":"owner"}]}}`, func(s *Service) error {
			got, err := s.GetHouseholdMember(context.Background(), "u-1")
			mustNoErr(t, err)
			eq(t, "Ann S", got.DisplayName)
			return nil
		})
	})

	t.Run("get household member missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdMembers", nil, `{"myHousehold":{"id":"hh-1","users":[]}}`, func(s *Service) error {
			_, err := s.GetHouseholdMember(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("get current user", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetMe", nil, `{"me":{"id":"u-1","email":"ann@example.com","name":"Ann","displayName":"Ann S","timezone":"America/Los_Angeles","householdRole":"owner","hasMfaOn":true,"createdAt":"2024-01-01"}}`, func(s *Service) error {
			got, err := s.GetCurrentUser(context.Background())
			mustNoErr(t, err)
			eq(t, "America/Los_Angeles", got.Timezone)
			return nil
		})
	})

	t.Run("get current user missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetMe", nil, `{"me":null}`, func(s *Service) error {
			_, err := s.GetCurrentUser(context.Background())
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update current user", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMe", map[string]any{"input": map[string]any{"timezone": "America/New_York"}}, `{"updateMe":{"user":{"id":"u-1","email":"ann@example.com","name":"Ann","displayName":"Ann S","timezone":"America/New_York","householdRole":"owner"},"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateCurrentUser(context.Background(), nil, strPtr("America/New_York"))
			mustNoErr(t, err)
			eq(t, "America/New_York", got.Timezone)
			return nil
		})
	})

	t.Run("update current user empty falls back to get", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetMe", nil, `{"me":{"id":"u-1","email":"a@b.c","name":"A","displayName":"A","timezone":"UTC","householdRole":"owner"}}`, func(s *Service) error {
			got, err := s.UpdateCurrentUser(context.Background(), nil, nil)
			mustNoErr(t, err)
			eq(t, "UTC", got.Timezone)
			return nil
		})
	})

	t.Run("update current user error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMe", map[string]any{"input": map[string]any{"displayName": "X"}}, `{"updateMe":{"user":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.UpdateCurrentUser(context.Background(), strPtr("X"), nil)
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update current user missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateMe", map[string]any{"input": map[string]any{"displayName": "X"}}, `{"updateMe":{"user":null,"errors":null}}`, func(s *Service) error {
			_, err := s.UpdateCurrentUser(context.Background(), strPtr("X"), nil)
			hasErr(t, err)
			return nil
		})
	})

	t.Run("get household preferences", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdPreferences", nil, `{"householdPreferences":{"id":"hp-1","newTransactionsNeedReview":true,"uncategorizedTransactionsNeedReview":false,"pendingTransactionsCanBeEdited":true},"budgetSystem":"flex"}`, func(s *Service) error {
			got, err := s.GetHouseholdPreferences(context.Background())
			mustNoErr(t, err)
			eq(t, "flex", got.BudgetSystem)
			return nil
		})
	})

	t.Run("get household preferences missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdPreferences", nil, `{"householdPreferences":null}`, func(s *Service) error {
			_, err := s.GetHouseholdPreferences(context.Background())
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update household preferences", func(t *testing.T) {
		var calls int
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				calls++
				if req.OperationName == "Common_UpdateHouseholdPreferences" {
					input, _ := req.Variables["input"].(map[string]any)
					if input["newTransactionsNeedReview"] != true {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"updateHouseholdPreferences":{"householdPreferences":{"id":"hp-1"}}}`)
				}
				return client.respond(result, `{"householdPreferences":{"id":"hp-1","newTransactionsNeedReview":true},"budgetSystem":"flex"}`)
			},
		}
		yes := true
		got, err := NewService(client).UpdateHouseholdPreferences(context.Background(), &HouseholdPreferencesUpdate{NewTransactionsNeedReview: &yes})
		mustNoErr(t, err)
		eq(t, "hp-1", got.ID)
		eq(t, 2, calls)
	})

	t.Run("update household preferences empty falls back to get", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetHouseholdPreferences", nil, `{"householdPreferences":{"id":"hp-1"},"budgetSystem":"flex"}`, func(s *Service) error {
			got, err := s.UpdateHouseholdPreferences(context.Background(), &HouseholdPreferencesUpdate{})
			mustNoErr(t, err)
			eq(t, "hp-1", got.ID)
			return nil
		})
	})

	t.Run("update household preferences missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateHouseholdPreferences", map[string]any{"input": map[string]any{"excludeBusinessFromBudget": true}}, `{"updateHouseholdPreferences":{"householdPreferences":null}}`, func(s *Service) error {
			no := true
			_, err := s.UpdateHouseholdPreferences(context.Background(), &HouseholdPreferencesUpdate{ExcludeBusinessFromBudget: &no})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("household error paths", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_GetMyHousehold", nil, func(s *Service) error {
			_, err := s.GetHousehold(context.Background())
			return err
		})
		runGraphQLErrorCase(t, "Common_GetHouseholdMembers", nil, func(s *Service) error {
			_, err := s.ListHouseholdMembers(context.Background())
			return err
		})
		runGraphQLErrorCase(t, "Common_GetMe", nil, func(s *Service) error {
			_, err := s.GetCurrentUser(context.Background())
			return err
		})
		runGraphQLErrorCase(t, "Common_GetHouseholdPreferences", nil, func(s *Service) error {
			_, err := s.GetHouseholdPreferences(context.Background())
			return err
		})
	})
}

func TestServiceReportPaths(t *testing.T) {
	reportDataPayload := `{"reports":[{"groupBy":{"date":"2026-01","category":{"id":"c-1","name":"Groceries"}},"summary":{"sum":-420.5,"avg":-35.0,"count":12,"max":-100.0,"sumIncome":0,"sumExpense":-420.5,"savings":0,"savingsRate":0,"first":"2026-01-02","last":"2026-01-28"}}],"aggregates":[{"summary":{"sum":-420.5,"count":12}}]}`

	t.Run("report data grouped by category", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_GetReportsData")
				if req.Variables["includeCategory"] != true {
					t.Fatalf("variables = %v", req.Variables)
				}
				if groups, ok := req.Variables["groupBy"].([]string); !ok || len(groups) != 1 || groups[0] != "category" {
					t.Fatalf("groupBy = %v", req.Variables["groupBy"])
				}
				return client.respond(result, reportDataPayload)
			},
		}
		got, err := NewService(client).GetReportData(context.Background(), &ReportDataOptions{StartDate: "2026-01-01", EndDate: "2026-01-31", GroupBy: "category"})
		mustNoErr(t, err)
		mustLen(t, got.Rows, 1)
		eq(t, "Groceries", got.Rows[0].Category)
		eq(t, 12, got.Summary.Count)
	})

	t.Run("report data merchant group and timeframe", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_GetReportsData")
				if req.Variables["includeMerchant"] != true || req.Variables["groupByTimeframe"] != "month" || req.Variables["sortBy"] != "sum" {
					t.Fatalf("variables = %v", req.Variables)
				}
				return client.respond(result, `{"reports":[{"groupBy":{"date":"2026-01","merchant":{"id":"m-1","name":"Store"}},"summary":{"sum":-10,"count":1}}],"aggregates":[]}`)
			},
		}
		got, err := NewService(client).GetReportData(context.Background(), &ReportDataOptions{GroupBy: "merchant", Timeframe: "month", SortBy: "sum", Search: "store"})
		mustNoErr(t, err)
		mustLen(t, got.Rows, 1)
		eq(t, "Store", got.Rows[0].Merchant)
		eq(t, 0, got.Summary.Count)
	})

	t.Run("report data category group label", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"reports":[{"groupBy":{"categoryGroup":{"id":"g-1","name":"Food"}},"summary":{"count":2}}],"aggregates":[]}`)
			},
		}
		got, err := NewService(client).GetReportData(context.Background(), &ReportDataOptions{GroupBy: "category-group"})
		mustNoErr(t, err)
		eq(t, "Food", got.Rows[0].CategoryGroup)
	})

	t.Run("report data invalid group", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t"})
		_, err := svc.GetReportData(context.Background(), &ReportDataOptions{GroupBy: "bogus"})
		hasErr(t, err)
	})

	t.Run("report data client error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_GetReportsData", map[string]any{
			"filters":              map[string]any{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}},
			"fillEmptyValues":      true,
			"includeCategory":      false,
			"includeCategoryGroup": false,
			"includeMerchant":      false,
		}, func(s *Service) error {
			_, err := s.GetReportData(context.Background(), &ReportDataOptions{})
			return err
		})
	})

	t.Run("list saved reports", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetReportConfigurations", nil, `{"reportConfigurations":[{"id":"r-1","displayName":"Monthly","reportView":{"timeframe":"month","chartType":"bar","dimensions":["category"]}},{"id":"r-2","displayName":"Bare","reportView":null}]}`, func(s *Service) error {
			got, err := s.ListSavedReports(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 2)
			eq(t, "Monthly", got[0].DisplayName)
			eq(t, "Bare", got[1].DisplayName)
			return nil
		})
	})

	t.Run("get saved report", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetReportConfigurations", nil, `{"reportConfigurations":[{"id":"r-1","displayName":"Monthly"}]}`, func(s *Service) error {
			got, err := s.GetSavedReport(context.Background(), "r-1")
			mustNoErr(t, err)
			eq(t, "Monthly", got.DisplayName)
			return nil
		})
	})

	t.Run("get saved report missing", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetReportConfigurations", nil, `{"reportConfigurations":[]}`, func(s *Service) error {
			_, err := s.GetSavedReport(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create saved report", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateReportConfiguration", map[string]any{"input": map[string]any{"displayName": "Q", "transactionFilters": map[string]any{}, "reportView": map[string]any{"dimensions": []string{"merchant"}, "timeframe": "quarter"}}}, `{"createReportConfiguration":{"reportConfiguration":{"id":"r-9","displayName":"Q"},"errors":null}}`, func(s *Service) error {
			got, err := s.CreateSavedReport(context.Background(), "Q", "merchant", "quarter")
			mustNoErr(t, err)
			eq(t, "r-9", got.ID)
			return nil
		})
	})

	t.Run("create saved report invalid group", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t"})
		_, err := svc.CreateSavedReport(context.Background(), "Q", "bogus", "")
		hasErr(t, err)
	})

	t.Run("create saved report error", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateReportConfiguration", map[string]any{"input": map[string]any{"displayName": "Q", "transactionFilters": map[string]any{}, "reportView": map[string]any{}}}, `{"createReportConfiguration":{"reportConfiguration":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.CreateSavedReport(context.Background(), "Q", "none", "")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create saved report missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateReportConfiguration", map[string]any{"input": map[string]any{"displayName": "Q", "transactionFilters": map[string]any{}, "reportView": map[string]any{}}}, `{"createReportConfiguration":{"reportConfiguration":null,"errors":null}}`, func(s *Service) error {
			_, err := s.CreateSavedReport(context.Background(), "Q", "none", "")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update saved report", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateReportConfiguration", map[string]any{"input": map[string]any{"id": "r-1", "displayName": "M2"}}, `{"updateReportConfiguration":{"reportConfiguration":{"id":"r-1","displayName":"M2"},"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateSavedReport(context.Background(), "r-1", "M2")
			mustNoErr(t, err)
			eq(t, "M2", got.DisplayName)
			return nil
		})
	})

	t.Run("update saved report error", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateReportConfiguration", map[string]any{"input": map[string]any{"id": "r-1", "displayName": "M2"}}, `{"updateReportConfiguration":{"reportConfiguration":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.UpdateSavedReport(context.Background(), "r-1", "M2")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update saved report missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateReportConfiguration", map[string]any{"input": map[string]any{"id": "r-1", "displayName": "M2"}}, `{"updateReportConfiguration":{"reportConfiguration":null,"errors":null}}`, func(s *Service) error {
			_, err := s.UpdateSavedReport(context.Background(), "r-1", "M2")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete saved report", func(t *testing.T) {
		runGraphQLCase(t, "Web_DeleteReportConfiguration", map[string]any{"id": "r-1"}, `{"deleteReportConfiguration":{"deleted":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteSavedReport(context.Background(), "r-1")
		})
	})

	t.Run("delete saved report error", func(t *testing.T) {
		runGraphQLCase(t, "Web_DeleteReportConfiguration", map[string]any{"id": "r-1"}, `{"deleteReportConfiguration":{"deleted":false,"errors":[{"message":"locked"}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteSavedReport(context.Background(), "r-1"))
			return nil
		})
	})

	t.Run("delete saved report not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Web_DeleteReportConfiguration", map[string]any{"id": "r-1"}, `{"deleteReportConfiguration":{"deleted":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.DeleteSavedReport(context.Background(), "r-1"))
			return nil
		})
	})
}

func TestServiceCategoryErrorPaths(t *testing.T) {
	boom := func(req *graphql.Request, result any) error {
		return fmt.Errorf("boom")
	}
	newBoomService := func() *Service { return NewService(&mockClient{token: "t", handler: boom}) }

	t.Run("create client error", func(t *testing.T) {
		_, err := newBoomService().CreateCategory(context.Background(), "x", "g", "")
		hasErr(t, err)
	})
	t.Run("delete client error", func(t *testing.T) {
		hasErr(t, newBoomService().DeleteCategory(context.Background(), "x", ""))
	})
	t.Run("get client error", func(t *testing.T) {
		_, err := newBoomService().GetCategory(context.Background(), "x")
		hasErr(t, err)
	})
	t.Run("reactivate client error", func(t *testing.T) {
		_, err := newBoomService().ReactivateCategory(context.Background(), "x")
		hasErr(t, err)
	})
	t.Run("reorder client error", func(t *testing.T) {
		_, err := newBoomService().ReorderCategory(context.Background(), "x", "g", 1)
		hasErr(t, err)
	})
	t.Run("create group client error", func(t *testing.T) {
		_, err := newBoomService().CreateCategoryGroup(context.Background(), "x", "expense")
		hasErr(t, err)
	})
	t.Run("delete group client error", func(t *testing.T) {
		hasErr(t, newBoomService().DeleteCategoryGroup(context.Background(), "x", ""))
	})
	t.Run("reorder group client error", func(t *testing.T) {
		_, err := newBoomService().ReorderCategoryGroup(context.Background(), "x", 1)
		hasErr(t, err)
	})
	t.Run("get category missing", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetEditCategory", map[string]any{"id": "nope"}, `{"category":null}`, func(s *Service) error {
			_, err := s.GetCategory(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("reactivate mutation error", func(t *testing.T) {
		runGraphQLCase(t, "Web_RestoreCategory", map[string]any{"id": "c1"}, `{"restoreCategory":{"category":null,"errors":[{"message":"gone"}]}}`, func(s *Service) error {
			_, err := s.ReactivateCategory(context.Background(), "c1")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("reactivate missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Web_RestoreCategory", map[string]any{"id": "c1"}, `{"restoreCategory":{"category":null,"errors":null}}`, func(s *Service) error {
			_, err := s.ReactivateCategory(context.Background(), "c1")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("reorder missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Web_UpdateCategoryOrder", map[string]any{"id": "c1", "categoryGroupId": "g1", "order": 1}, `{"updateCategoryOrderInCategoryGroup":{"category":null}}`, func(s *Service) error {
			_, err := s.ReorderCategory(context.Background(), "c1", "g1", 1)
			hasErr(t, err)
			return nil
		})
	})
	t.Run("create group missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateCategoryGroup", map[string]any{"input": map[string]any{"name": "P", "type": "expense"}}, `{"createCategoryGroup":{"categoryGroup":null}}`, func(s *Service) error {
			_, err := s.CreateCategoryGroup(context.Background(), "P", "expense")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("delete group payload error", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteCategoryGroup", map[string]any{"id": "g1"}, `{"deleteCategoryGroup":{"deleted":false,"errors":[{"message":"used"}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteCategoryGroup(context.Background(), "g1", ""))
			return nil
		})
	})
	t.Run("delete group not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteCategoryGroup", map[string]any{"id": "g1"}, `{"deleteCategoryGroup":{"deleted":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.DeleteCategoryGroup(context.Background(), "g1", ""))
			return nil
		})
	})
	t.Run("create payload error", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateCategory", map[string]any{"input": map[string]any{"name": "x", "group": "g"}}, `{"createCategory":{"category":null,"errors":[{"message":"taken"}]}}`, func(s *Service) error {
			_, err := s.CreateCategory(context.Background(), "x", "g", "")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("create missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Web_CreateCategory", map[string]any{"input": map[string]any{"name": "x", "group": "g"}}, `{"createCategory":{"category":null,"errors":null}}`, func(s *Service) error {
			_, err := s.CreateCategory(context.Background(), "x", "g", "")
			hasErr(t, err)
			return nil
		})
	})
	t.Run("delete not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Web_DeleteCategory", map[string]any{"id": "c1"}, `{"deleteCategory":{"deleted":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.DeleteCategory(context.Background(), "c1", ""))
			return nil
		})
	})
}

func TestServiceRecurringStreamPaths(t *testing.T) {
	streamPayload := `{"stream":{"id":"rs-1","name":"Netflix","frequency":"monthly","amount":15.99,"baseDate":"2026-01-15","isActive":true,"isApproximate":false,"merchant":{"id":"m-1","name":"Netflix"}},"nextForecastedTransaction":{"date":"2026-06-15","amount":15.99},"category":{"id":"cat-1","name":"Entertainment"},"account":{"id":"acc-1","displayName":"Checking"}}`

	t.Run("list streams", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetAllRecurringTransactionItems", nil, `{"recurringTransactionStreams":[`+streamPayload+`]}`, func(s *Service) error {
			_ = s
			return nil
		})
	})

	t.Run("list streams decoded", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_GetAllRecurringTransactionItems")
				return client.respond(result, `{"recurringTransactionStreams":[`+streamPayload+`]}`)
			},
		}
		got, err := NewService(client).ListRecurringStreams(context.Background())
		mustNoErr(t, err)
		mustLen(t, got, 1)
		eq(t, "rs-1", got[0].ID)
		eq(t, "Netflix", got[0].MerchantName)
		eq(t, "2026-06-15", got[0].NextDate)
	})

	t.Run("list streams skips nil", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"recurringTransactionStreams":[{"stream":null},`+streamPayload+`]}`)
			},
		}
		got, err := NewService(client).ListRecurringStreams(context.Background())
		mustNoErr(t, err)
		mustLen(t, got, 1)
	})

	t.Run("get stream", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"recurringTransactionStreams":[`+streamPayload+`]}`)
			},
		}
		got, err := NewService(client).GetRecurringStream(context.Background(), "rs-1")
		mustNoErr(t, err)
		eq(t, "monthly", got.Frequency)
	})

	t.Run("get stream missing", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"recurringTransactionStreams":[]}`)
			},
		}
		_, err := NewService(client).GetRecurringStream(context.Background(), "nope")
		hasErr(t, err)
	})

	t.Run("summary", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetAggregatedRecurringItems", map[string]any{"startDate": "2026-05-01", "endDate": "2026-07-31", "filters": map[string]any{}}, `{"aggregatedRecurringItems":{"aggregatedSummary":{"expense":{"completed":100,"remaining":50,"total":150,"count":3},"income":{"completed":2000,"remaining":0,"total":2000}}}}`, func(s *Service) error {
			got, err := s.GetRecurringSummary(context.Background(), "2026-05-01", "2026-07-31")
			mustNoErr(t, err)
			eq(t, 150.0, got.ExpenseTotal)
			eq(t, 3, got.ExpenseCount)
			eq(t, 2000.0, got.IncomeTotal)
			return nil
		})
	})

	t.Run("summary nil aggregates", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"aggregatedRecurringItems":null}`)
			},
		}
		got, err := NewService(client).GetRecurringSummary(context.Background(), "2026-05-01", "2026-07-31")
		mustNoErr(t, err)
		eq(t, 0.0, got.ExpenseTotal)
	})

	t.Run("summary nil summary", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"aggregatedRecurringItems":{"aggregatedSummary":null}}`)
			},
		}
		_, err := NewService(client).GetRecurringSummary(context.Background(), "2026-05-01", "2026-07-31")
		mustNoErr(t, err)
	})

	t.Run("create stream", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_RecurringUpdateMerchant" {
					input, _ := req.Variables["input"].(map[string]any)
					recurrence, _ := input["recurrence"].(map[string]any)
					if input["merchantId"] != "m-1" || recurrence["frequency"] != "monthly" || recurrence["amount"] != 15.99 || recurrence["isRecurring"] != true {
						t.Fatalf("input = %v", input)
					}
					if _, ok := recurrence["baseDate"]; ok {
						t.Fatalf("create should not send baseDate: %v", input)
					}
					return client.respond(result, `{"updateMerchant":{"merchant":{"id":"m-1"},"errors":null}}`)
				}
				return client.respond(result, `{"recurringTransactionStreams":[`+streamPayload+`]}`)
			},
		}
		freq := "monthly"
		amount := 15.99
		got, err := NewService(client).CreateRecurringStream(context.Background(), "m-1", &RecurringStreamInput{Frequency: &freq, Amount: &amount})
		mustNoErr(t, err)
		eq(t, "rs-1", got.ID)
	})

	t.Run("create stream not found after", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_RecurringUpdateMerchant" {
					return client.respond(result, `{"updateMerchant":{"merchant":{"id":"m-9"},"errors":null}}`)
				}
				return client.respond(result, `{"recurringTransactionStreams":[]}`)
			},
		}
		_, err := NewService(client).CreateRecurringStream(context.Background(), "m-9", &RecurringStreamInput{})
		hasErr(t, err)
	})

	t.Run("create stream mutation error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_RecurringUpdateMerchant", map[string]any{"input": map[string]any{"merchantId": "m-1", "recurrence": map[string]any{"isRecurring": true}}}, func(s *Service) error {
			_, err := s.CreateRecurringStream(context.Background(), "m-1", &RecurringStreamInput{})
			return err
		})
	})

	t.Run("create stream payload error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"updateMerchant":{"errors":[{"message":"bad"}]}}`)
			},
		}
		_, err := NewService(client).CreateRecurringStream(context.Background(), "m-1", &RecurringStreamInput{})
		hasErr(t, err)
	})

	t.Run("update stream", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_RecurringUpdateMerchant" {
					input, _ := req.Variables["input"].(map[string]any)
					recurrence, _ := input["recurrence"].(map[string]any)
					if input["merchantId"] != "m-1" || recurrence["amount"] != 19.99 || recurrence["isActive"] != false || recurrence["frequency"] != "monthly" || recurrence["baseDate"] != "2026-01-15" {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"updateMerchant":{"merchant":{"id":"m-1"},"errors":null}}`)
				}
				return client.respond(result, `{"recurringTransactionStreams":[`+streamPayload+`]}`)
			},
		}
		amount := 19.99
		active := false
		got, err := NewService(client).UpdateRecurringStream(context.Background(), "rs-1", &RecurringStreamInput{Amount: &amount, IsActive: &active})
		mustNoErr(t, err)
		eq(t, "rs-1", got.ID)
	})

	t.Run("update stream without merchant", func(t *testing.T) {
		bare := strings.Replace(streamPayload, `"merchant":{"id":"m-1","name":"Netflix"}`, `"merchant":null`, 1)
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"recurringTransactionStreams":[`+bare+`]}`)
			},
		}
		_, err := NewService(client).UpdateRecurringStream(context.Background(), "rs-1", &RecurringStreamInput{})
		hasErr(t, err)
	})

	t.Run("update stream missing", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"recurringTransactionStreams":[]}`)
			},
		}
		_, err := NewService(client).UpdateRecurringStream(context.Background(), "nope", &RecurringStreamInput{})
		hasErr(t, err)
	})

	t.Run("remove stream", func(t *testing.T) {
		runGraphQLCase(t, "Common_MarkAsNotRecurring", map[string]any{"streamId": "rs-1"}, `{"markStreamAsNotRecurring":{"success":true,"errors":null}}`, func(s *Service) error {
			return s.RemoveRecurringStream(context.Background(), "rs-1")
		})
	})

	t.Run("remove stream error", func(t *testing.T) {
		runGraphQLCase(t, "Common_MarkAsNotRecurring", map[string]any{"streamId": "rs-1"}, `{"markStreamAsNotRecurring":{"success":false,"errors":[{"message":"locked"}]}}`, func(s *Service) error {
			hasErr(t, s.RemoveRecurringStream(context.Background(), "rs-1"))
			return nil
		})
	})

	t.Run("remove stream not removed", func(t *testing.T) {
		runGraphQLCase(t, "Common_MarkAsNotRecurring", map[string]any{"streamId": "rs-1"}, `{"markStreamAsNotRecurring":{"success":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.RemoveRecurringStream(context.Background(), "rs-1"))
			return nil
		})
	})

	t.Run("stream error paths", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_GetAllRecurringTransactionItems", map[string]any{"filters": map[string]any{}, "includePending": true, "includeLiabilities": true}, func(s *Service) error {
			_, err := s.ListRecurringStreams(context.Background())
			return err
		})
	})
}

func TestServiceInvestmentGapPaths(t *testing.T) {
	holdingNode := `{"id":"h-1","quantity":10,"totalValue":1500.0,"security":{"id":"s-1","name":"Apple Inc","ticker":"AAPL","currentPrice":150.0},"holdings":[{"id":"h-1","name":"Apple","ticker":"AAPL","quantity":10,"value":1500.0,"account":{"id":"a-1","displayName":"Brokerage"}}]}`

	t.Run("list investment accounts", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetInvestmentsAccounts", nil, `{"accounts":[{"id":"a-1","displayName":"Brokerage","includeInNetWorth":true}]}`, func(s *Service) error {
			got, err := s.ListInvestmentAccounts(context.Background())
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "Brokerage", got[0].DisplayName)
			return nil
		})
	})

	t.Run("list holding details", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":`+holdingNode+`}]}}}`, func(s *Service) error {
			got, err := s.ListHoldingDetails(context.Background(), nil)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "AAPL", got[0].Ticker)
			eq(t, "s-1", got[0].SecurityID)
			eq(t, "Brokerage", got[0].AccountName)
			return nil
		})
	})

	t.Run("list holding details with accounts", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{"accountIds": []string{"a-1"}}}, `{"portfolio":{"aggregateHoldings":{"edges":[]}}}`, func(s *Service) error {
			got, err := s.ListHoldingDetails(context.Background(), []string{"a-1"})
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("list holding details nil portfolio", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, `{"portfolio":null}`, func(s *Service) error {
			got, err := s.ListHoldingDetails(context.Background(), nil)
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("list holding details skips nil nodes", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":null},{"node":`+holdingNode+`}]}}}`, func(s *Service) error {
			got, err := s.ListHoldingDetails(context.Background(), nil)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			return nil
		})
	})

	t.Run("get holding detail", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":`+holdingNode+`}]}}}`, func(s *Service) error {
			got, err := s.GetHoldingDetail(context.Background(), "h-1")
			mustNoErr(t, err)
			eq(t, "h-1", got.ID)
			return nil
		})
	})

	t.Run("get holding detail missing", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, `{"portfolio":{"aggregateHoldings":{"edges":[]}}}`, func(s *Service) error {
			_, err := s.GetHoldingDetail(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("search securities", func(t *testing.T) {
		runGraphQLCase(t, "Web_SearchSecurities", map[string]any{"search": "Apple", "limit": 20, "orderByPopularity": true}, `{"securities":[{"id":"s-1","name":"Apple Inc","ticker":"AAPL","type":"stock","typeDisplay":"Stock","currentPrice":150.0},null]}`, func(s *Service) error {
			got, err := s.SearchSecurities(context.Background(), "Apple", 0)
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "AAPL", got[0].Ticker)
			return nil
		})
	})

	t.Run("get security", func(t *testing.T) {
		runGraphQLCase(t, "GetHoldingDetailsFormSecurityDetails", map[string]any{"id": "s-1"}, `{"security":{"id":"s-1","name":"Apple Inc","ticker":"AAPL"}}`, func(s *Service) error {
			got, err := s.GetSecurity(context.Background(), "s-1")
			mustNoErr(t, err)
			eq(t, "Apple Inc", got.Name)
			return nil
		})
	})

	t.Run("get security missing", func(t *testing.T) {
		runGraphQLCase(t, "GetHoldingDetailsFormSecurityDetails", map[string]any{"id": "nope"}, `{"security":null}`, func(s *Service) error {
			_, err := s.GetSecurity(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create manual holding", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_CreateManualHolding" {
					input, _ := req.Variables["input"].(map[string]any)
					if input["accountId"] != "a-1" || input["securityId"] != "s-1" || input["quantity"] != 10.0 {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"createManualHolding":{"holding":{"id":"h-9","ticker":"AAPL"},"errors":null}}`)
				}
				return client.respond(result, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h-9","quantity":10,"totalValue":1500.0,"security":{"id":"s-1"},"holdings":[{"id":"h-9","name":"Apple","ticker":"AAPL","quantity":10,"value":1500.0,"account":{"id":"a-1","displayName":"Brokerage"}}]}}]}}}`)
			},
		}
		got, err := NewService(client).CreateManualHolding(context.Background(), &ManualHoldingInput{AccountID: "a-1", SecurityID: "s-1", Quantity: 10})
		mustNoErr(t, err)
		eq(t, "h-9", got.ID)
	})

	t.Run("create manual holding with cost basis", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "Common_CreateManualHolding":
					return client.respond(result, `{"createManualHolding":{"holding":{"id":"h-9","ticker":"AAPL"},"errors":null}}`)
				case "Common_UpdateHolding":
					input, _ := req.Variables["input"].(map[string]any)
					if input["userCostBasis"] != 1400.0 {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"updateHolding":{"holding":{"id":"h-9"},"errors":null}}`)
				default:
					return client.respond(result, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h-9","quantity":10,"totalValue":1500.0,"holdings":[{"id":"h-9","quantity":10,"value":1500.0}]}}]}}}`)
				}
			},
		}
		basis := 1400.0
		got, err := NewService(client).CreateManualHolding(context.Background(), &ManualHoldingInput{AccountID: "a-1", SecurityID: "s-1", Quantity: 10, CostBasis: &basis})
		mustNoErr(t, err)
		eq(t, "h-9", got.ID)
	})

	t.Run("create manual holding error", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateManualHolding", map[string]any{"input": map[string]any{"accountId": "a-1", "securityId": "s-1", "quantity": 10.0}}, `{"createManualHolding":{"holding":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.CreateManualHolding(context.Background(), &ManualHoldingInput{AccountID: "a-1", SecurityID: "s-1", Quantity: 10})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create manual holding missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateManualHolding", map[string]any{"input": map[string]any{"accountId": "a-1", "securityId": "s-1", "quantity": 10.0}}, `{"createManualHolding":{"holding":null,"errors":null}}`, func(s *Service) error {
			_, err := s.CreateManualHolding(context.Background(), &ManualHoldingInput{AccountID: "a-1", SecurityID: "s-1", Quantity: 10})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update manual holding", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_UpdateHolding" {
					input, _ := req.Variables["input"].(map[string]any)
					if input["quantity"] != 20.0 || input["securityType"] != "stock" {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"updateHolding":{"holding":{"id":"h-1"},"errors":null}}`)
				}
				return client.respond(result, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":`+holdingNode+`}]}}}`)
			},
		}
		qty := 20.0
		stype := "stock"
		got, err := NewService(client).UpdateManualHolding(context.Background(), "h-1", &ManualHoldingUpdate{Quantity: &qty, SecurityType: &stype})
		mustNoErr(t, err)
		eq(t, "h-1", got.ID)
	})

	t.Run("update manual holding error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateHolding", map[string]any{"input": map[string]any{"id": "h-1"}}, `{"updateHolding":{"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.UpdateManualHolding(context.Background(), "h-1", &ManualHoldingUpdate{})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete manual holding", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHolding", map[string]any{"id": "h-1"}, `{"deleteHolding":{"deleted":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteManualHolding(context.Background(), "h-1")
		})
	})

	t.Run("delete manual holding error", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHolding", map[string]any{"id": "h-1"}, `{"deleteHolding":{"deleted":false,"errors":[{"message":"locked"}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteManualHolding(context.Background(), "h-1"))
			return nil
		})
	})

	t.Run("delete manual holding not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteHolding", map[string]any{"id": "h-1"}, `{"deleteHolding":{"deleted":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.DeleteManualHolding(context.Background(), "h-1"))
			return nil
		})
	})

	t.Run("investment error paths", func(t *testing.T) {
		runGraphQLErrorCase(t, "Web_GetInvestmentsAccounts", nil, func(s *Service) error {
			_, err := s.ListInvestmentAccounts(context.Background())
			return err
		})
		runGraphQLErrorCase(t, "Web_GetHoldings", map[string]any{"input": map[string]any{}}, func(s *Service) error {
			_, err := s.ListHoldingDetails(context.Background(), nil)
			return err
		})
		runGraphQLErrorCase(t, "Web_SearchSecurities", map[string]any{"search": "x", "limit": 20, "orderByPopularity": true}, func(s *Service) error {
			_, err := s.SearchSecurities(context.Background(), "x", 0)
			return err
		})
		runGraphQLErrorCase(t, "GetHoldingDetailsFormSecurityDetails", map[string]any{"id": "s-1"}, func(s *Service) error {
			_, err := s.GetSecurity(context.Background(), "s-1")
			return err
		})
	})
}

func TestServiceTransactionGapErrorPaths(t *testing.T) {
	t.Run("get attachment client error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Mobile_GetAttachmentDetails", map[string]any{"attachmentId": "att-1"}, func(s *Service) error {
			_, err := s.GetTransactionAttachment(context.Background(), "att-1")
			return err
		})
	})

	t.Run("get attachment missing", func(t *testing.T) {
		runGraphQLCase(t, "Mobile_GetAttachmentDetails", map[string]any{"attachmentId": "nope"}, `{"transactionAttachment":null}`, func(s *Service) error {
			_, err := s.GetTransactionAttachment(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete attachment client error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Web_TransactionDrawerDeleteAttachment", map[string]any{"id": "att-1"}, func(s *Service) error {
			return s.DeleteTransactionAttachment(context.Background(), "att-1")
		})
	})

	t.Run("delete attachment not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Web_TransactionDrawerDeleteAttachment", map[string]any{"id": "att-1"}, `{"deleteTransactionAttachment":{"deleted":false}}`, func(s *Service) error {
			hasErr(t, s.DeleteTransactionAttachment(context.Background(), "att-1"))
			return nil
		})
	})

	t.Run("link goal client error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_LinkTransactionToGoal", map[string]any{"input": map[string]any{"transactionId": "tx-1", "goalId": "g-1"}}, func(s *Service) error {
			return s.LinkTransactionToGoal(context.Background(), "tx-1", "g-1", "")
		})
	})

	t.Run("link goal payload error", func(t *testing.T) {
		runGraphQLCase(t, "Common_LinkTransactionToGoal", map[string]any{"input": map[string]any{"transactionId": "tx-1"}}, `{"linkTransactionToGoal":{"errors":[{"message":"bad goal"}]}}`, func(s *Service) error {
			hasErr(t, s.LinkTransactionToGoal(context.Background(), "tx-1", "", ""))
			return nil
		})
	})
}

func TestServiceBudgetGapErrorPaths(t *testing.T) {
	boomHandler := func(req *graphql.Request, result any) error {
		return fmt.Errorf("boom")
	}
	t.Run("reset client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		hasErr(t, svc.ResetBudget(context.Background(), "2026-05-01", false, nil))
	})
	t.Run("settings client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		_, err := svc.GetBudgetSettings(context.Background())
		hasErr(t, err)
	})
	t.Run("flex rollover client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		_, err := svc.GetFlexRolloverSettings(context.Background())
		hasErr(t, err)
	})
	t.Run("set group client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		hasErr(t, svc.SetBudgetGroup(context.Background(), "g1", 1, "2026-05-01"))
	})
	t.Run("set group payload error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateBudgetItem", map[string]any{"input": map[string]any{"categoryGroupId": "g1", "amount": 1.0, "timeframe": "month", "startDate": "2026-05-01", "applyToFuture": false}}, `{"updateOrCreateBudgetItem":{"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			hasErr(t, s.SetBudgetGroup(context.Background(), "g1", 1, "2026-05-01"))
			return nil
		})
	})
	t.Run("create client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		hasErr(t, svc.CreateBudget(context.Background(), "2026-05-01"))
	})
	t.Run("create payload error", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateBudgetForHousehold", map[string]any{"input": map[string]any{"startDate": "2026-05-01", "timeframe": "month"}}, `{"createBudget":{"errors":[{"message":"exists"}]}}`, func(s *Service) error {
			hasErr(t, s.CreateBudget(context.Background(), "2026-05-01"))
			return nil
		})
	})
	t.Run("clear client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		hasErr(t, svc.ClearBudget(context.Background(), "2026-05-01"))
	})
	t.Run("clear payload error", func(t *testing.T) {
		runGraphQLCase(t, "Web_ClearAllMutation", map[string]any{"input": map[string]any{"startDate": "2026-05-01"}}, `{"clearBudget":{"errors":[{"message":"locked"}]}}`, func(s *Service) error {
			hasErr(t, s.ClearBudget(context.Background(), "2026-05-01"))
			return nil
		})
	})
	t.Run("reset rollover client error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		hasErr(t, svc.ResetBudgetRollover(context.Background(), &ResetRolloverOptions{StartMonth: "2026-05-01"}))
	})
	t.Run("reset rollover payload error", func(t *testing.T) {
		runGraphQLCase(t, "Web_ResetRolloverMutation", map[string]any{"input": map[string]any{"startMonth": "2026-05-01", "categoryGroupId": "g1", "startingBalance": 10.0}}, `{"resetBudgetRollover":{"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			bal := 10.0
			hasErr(t, s.ResetBudgetRollover(context.Background(), &ResetRolloverOptions{StartMonth: "2026-05-01", CategoryGroupID: "g1", StartingBalance: &bal}))
			return nil
		})
	})
}

func TestServiceGoalGapPaths(t *testing.T) {
	goalPayload := `{"id":"goal-1","type":"custom","name":"Emergency","status":"active","progress":0.5,"currentBalance":5000,"targetDate":"2027-01-01","targetAmount":10000,"plannedMonthlyContribution":200,"currentMonthPlannedContributionAmount":200,"spendingTotal":0,"netContribution":5000,"estimatedMonthsUntilCompletion":25,"forecastedCompletionDate":"2028-01-01","isSinkingFund":false,"priority":1}`
	eventPayload := `{"id":"ge-1","date":"2026-05-01","amount":200,"type":"contribution","notes":"may","goal":{"id":"goal-1","name":"Emergency"},"account":{"id":"acc-1","displayName":"Checking"}}`
	boomHandler := func(req *graphql.Request, result any) error {
		return fmt.Errorf("boom")
	}

	t.Run("get goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoal", map[string]any{"id": "goal-1"}, `{"savingsGoal":`+goalPayload+`}`, func(s *Service) error {
			got, err := s.GetGoal(context.Background(), "goal-1")
			mustNoErr(t, err)
			eq(t, "Emergency", got.Name)
			return nil
		})
	})

	t.Run("get goal missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoal", map[string]any{"id": "nope"}, `{"savingsGoal":null}`, func(s *Service) error {
			_, err := s.GetGoal(context.Background(), "nope")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create goal", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_CreateSavingsGoals" {
					return client.respond(result, `{"createSavingsGoals":{"savingsGoals":[{"id":"goal-1","type":"custom"}],"errors":null}}`)
				}
				return client.respond(result, `{"savingsGoal":`+goalPayload+`}`)
			},
		}
		target := 10000.0
		got, err := NewService(client).CreateGoal(context.Background(), &GoalInput{Name: "Emergency", Type: "custom", TargetAmount: &target})
		mustNoErr(t, err)
		eq(t, "goal-1", got.ID)
	})

	t.Run("create goal error", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateSavingsGoals", map[string]any{"input": map[string]any{"goals": []any{map[string]any{"name": "x"}}}}, `{"createSavingsGoals":{"savingsGoals":[],"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.CreateGoal(context.Background(), &GoalInput{Name: "x"})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("create goal missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateSavingsGoals", map[string]any{"input": map[string]any{"goals": []any{map[string]any{"name": "x"}}}}, `{"createSavingsGoals":{"savingsGoals":[],"errors":null}}`, func(s *Service) error {
			_, err := s.CreateGoal(context.Background(), &GoalInput{Name: "x"})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update goal", func(t *testing.T) {
		name := "New Name"
		runGraphQLCase(t, "Common_UpdateSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1", "name": "New Name"}}, `{"updateSavingsGoal":{"savingsGoal":`+goalPayload+`,"errors":null}}`, func(s *Service) error {
			got, err := s.UpdateGoal(context.Background(), "goal-1", &GoalUpdate{Name: &name})
			mustNoErr(t, err)
			eq(t, "goal-1", got.ID)
			return nil
		})
	})

	t.Run("update goal all fields", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				input, _ := req.Variables["input"].(map[string]any)
				for _, k := range []string{"id", "name", "type", "targetAmount", "targetDate", "budgetToApplyToFutureMonths", "isSinkingFund", "priority"} {
					if _, ok := input[k]; !ok {
						t.Fatalf("input missing %q: %v", k, input)
					}
				}
				return client.respond(result, `{"updateSavingsGoal":{"savingsGoal":`+goalPayload+`,"errors":null}}`)
			},
		}
		name, gtype, date := "N", "custom", "2027-01-01"
		amount, monthly := 1.0, 2.0
		sinking := true
		prio := 3
		_, err := NewService(client).UpdateGoal(context.Background(), "goal-1", &GoalUpdate{Name: &name, Type: &gtype, TargetAmount: &amount, TargetDate: &date, MonthlyContribution: &monthly, SinkingFund: &sinking, Priority: &prio})
		mustNoErr(t, err)
	})

	t.Run("update goal error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"updateSavingsGoal":{"savingsGoal":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.UpdateGoal(context.Background(), "goal-1", &GoalUpdate{})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update goal missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"updateSavingsGoal":{"savingsGoal":null,"errors":null}}`, func(s *Service) error {
			_, err := s.UpdateGoal(context.Background(), "goal-1", &GoalUpdate{})
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"deleteSavingsGoal":{"success":true,"errors":null}}`, func(s *Service) error {
			return s.DeleteGoal(context.Background(), "goal-1")
		})
	})

	t.Run("delete goal error", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"deleteSavingsGoal":{"success":false,"errors":[{"message":"locked"}]}}`, func(s *Service) error {
			hasErr(t, s.DeleteGoal(context.Background(), "goal-1"))
			return nil
		})
	})

	t.Run("delete goal not deleted", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"deleteSavingsGoal":{"success":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.DeleteGoal(context.Background(), "goal-1"))
			return nil
		})
	})

	t.Run("archive goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_ArchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"archiveSavingsGoal":{"savingsGoal":`+goalPayload+`,"errors":null}}`, func(s *Service) error {
			got, err := s.ArchiveGoal(context.Background(), "goal-1")
			mustNoErr(t, err)
			eq(t, "goal-1", got.ID)
			return nil
		})
	})

	t.Run("archive goal error", func(t *testing.T) {
		runGraphQLCase(t, "Common_ArchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"archiveSavingsGoal":{"savingsGoal":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.ArchiveGoal(context.Background(), "goal-1")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("archive goal missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_ArchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"archiveSavingsGoal":{"savingsGoal":null,"errors":null}}`, func(s *Service) error {
			_, err := s.ArchiveGoal(context.Background(), "goal-1")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("restore goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_UnarchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"unarchiveSavingsGoal":{"savingsGoal":`+goalPayload+`,"errors":null}}`, func(s *Service) error {
			got, err := s.RestoreGoal(context.Background(), "goal-1")
			mustNoErr(t, err)
			eq(t, "goal-1", got.ID)
			return nil
		})
	})

	t.Run("restore goal error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UnarchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"unarchiveSavingsGoal":{"savingsGoal":null,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			_, err := s.RestoreGoal(context.Background(), "goal-1")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("restore goal missing payload", func(t *testing.T) {
		runGraphQLCase(t, "Common_UnarchiveSavingsGoal", map[string]any{"input": map[string]any{"id": "goal-1"}}, `{"unarchiveSavingsGoal":{"savingsGoal":null,"errors":null}}`, func(s *Service) error {
			_, err := s.RestoreGoal(context.Background(), "goal-1")
			hasErr(t, err)
			return nil
		})
	})

	t.Run("update priorities", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoalsPriorities", map[string]any{"input": map[string]any{"goals": []map[string]any{{"id": "goal-1", "priority": 0}}}}, `{"updateSavingsGoalsPriorities":{"goals":[{"id":"goal-1","priority":0}],"errors":null}}`, func(s *Service) error {
			return s.UpdateGoalPriorities(context.Background(), []string{"goal-1"})
		})
	})

	t.Run("update priorities error", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoalsPriorities", map[string]any{"input": map[string]any{"goals": []map[string]any{{"id": "goal-1", "priority": 0}}}}, `{"updateSavingsGoalsPriorities":{"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			hasErr(t, s.UpdateGoalPriorities(context.Background(), []string{"goal-1"}))
			return nil
		})
	})

	t.Run("link goal account", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_CreateSavingsGoalAccountInitialContributions" {
					input, _ := req.Variables["input"].(map[string]any)
					if input["accountId"] != "acc-1" {
						t.Fatalf("input = %v", input)
					}
					return client.respond(result, `{"createGoalAccountInitialContributions":{"userNotice":null,"errors":null}}`)
				}
				return client.respond(result, `{"savingsGoal":`+goalPayload+`}`)
			},
		}
		amount := 100.0
		got, err := NewService(client).LinkGoalAccount(context.Background(), "goal-1", "acc-1", &amount, false)
		mustNoErr(t, err)
		eq(t, "goal-1", got.ID)
	})

	t.Run("link goal account payload error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"createGoalAccountInitialContributions":{"errors":[{"message":"bad"}]}}`)
			},
		}
		_, err := NewService(client).LinkGoalAccount(context.Background(), "goal-1", "acc-1", nil, true)
		hasErr(t, err)
	})

	t.Run("unlink goal account", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_CreateSavingsGoalAccountInitialContributions" {
					return client.respond(result, `{"createGoalAccountInitialContributions":{"errors":null}}`)
				}
				return client.respond(result, `{"savingsGoal":`+goalPayload+`}`)
			},
		}
		got, err := NewService(client).UnlinkGoalAccount(context.Background(), "goal-1", "acc-1")
		mustNoErr(t, err)
		eq(t, "goal-1", got.ID)
	})

	t.Run("list goal events", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoalEvents", map[string]any{"id": "goal-1"}, `{"savingsGoal":{"id":"goal-1","goalEvents":[`+eventPayload+`,null]}}`, func(s *Service) error {
			got, err := s.ListGoalEvents(context.Background(), "goal-1")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "ge-1", got[0].ID)
			eq(t, "Checking", got[0].AccountName)
			return nil
		})
	})

	t.Run("list goal events nil goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoalEvents", map[string]any{"id": "goal-1"}, `{"savingsGoal":null}`, func(s *Service) error {
			got, err := s.ListGoalEvents(context.Background(), "goal-1")
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("contribute to goal", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_ContributeToSavingsGoal" {
					return client.respond(result, `{"createSavingsGoalContribution":{"userNotice":null,"goalEvent":{"id":"ge-1"}}}`)
				}
				return client.respond(result, `{"savingsGoal":{"id":"goal-1","goalEvents":[`+eventPayload+`]}}`)
			},
		}
		got, err := NewService(client).ContributeToGoal(context.Background(), "goal-1", "acc-1", 200, nil, nil)
		mustNoErr(t, err)
		eq(t, "ge-1", got.ID)
	})

	t.Run("contribute event not refetched", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_ContributeToSavingsGoal" {
					return client.respond(result, `{"createSavingsGoalContribution":{"goalEvent":{"id":"ge-9"}}}`)
				}
				return client.respond(result, `{"savingsGoal":{"id":"goal-1","goalEvents":[]}}`)
			},
		}
		_, err := NewService(client).ContributeToGoal(context.Background(), "goal-1", "acc-1", 200, nil, nil)
		hasErr(t, err)
	})

	t.Run("contribute missing event", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"createSavingsGoalContribution":{"goalEvent":null}}`)
			},
		}
		_, err := NewService(client).ContributeToGoal(context.Background(), "goal-1", "acc-1", 200, nil, nil)
		hasErr(t, err)
	})

	t.Run("withdraw from goal", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "Common_WithdrawFromSavingsGoal" {
					return client.respond(result, `{"createSavingsGoalWithdrawal":{"goalEvent":{"id":"ge-1"}}}`)
				}
				return client.respond(result, `{"savingsGoal":{"id":"goal-1","goalEvents":[`+eventPayload+`]}}`)
			},
		}
		got, err := NewService(client).WithdrawFromGoal(context.Background(), "goal-1", "acc-1", 50, nil, nil)
		mustNoErr(t, err)
		eq(t, "ge-1", got.ID)
	})

	t.Run("update goal event", func(t *testing.T) {
		notes := "june"
		runGraphQLCase(t, "Common_UpdateSavingsGoalEvent", map[string]any{"input": map[string]any{"eventId": "ge-1", "notes": "june"}}, `{"updateGoalEvent":{"goalEvent":`+eventPayload+`}}`, func(s *Service) error {
			got, err := s.UpdateGoalEvent(context.Background(), "ge-1", nil, &notes)
			mustNoErr(t, err)
			eq(t, "ge-1", got.ID)
			return nil
		})
	})

	t.Run("update goal event missing", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoalEvent", map[string]any{"input": map[string]any{"eventId": "ge-1"}}, `{"updateGoalEvent":{"goalEvent":null}}`, func(s *Service) error {
			_, err := s.UpdateGoalEvent(context.Background(), "ge-1", nil, nil)
			hasErr(t, err)
			return nil
		})
	})

	t.Run("delete goal event", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteSavingsGoalEvent", map[string]any{"input": map[string]any{"eventId": "ge-1"}}, `{"deleteGoalEvent":{"success":true}}`, func(s *Service) error {
			return s.DeleteGoalEvent(context.Background(), "ge-1")
		})
	})

	t.Run("delete goal event failure", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteSavingsGoalEvent", map[string]any{"input": map[string]any{"eventId": "ge-1"}}, `{"deleteGoalEvent":{"success":false}}`, func(s *Service) error {
			hasErr(t, s.DeleteGoalEvent(context.Background(), "ge-1"))
			return nil
		})
	})

	t.Run("goal budget amounts", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoalBudgetAmounts", map[string]any{"goalId": "goal-1", "startMonth": "2026-05-01", "endMonth": "2026-05-31"}, `{"savingsGoal":{"id":"goal-1","monthlyBudgetAmounts":[{"id":"mba-1","month":"2026-05-01","plannedAmount":200,"actualAmount":100,"remainingAmount":100}]}}`, func(s *Service) error {
			got, err := s.GetGoalBudgetAmounts(context.Background(), "goal-1", "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustLen(t, got, 1)
			eq(t, "2026-05-01", got[0].Month)
			return nil
		})
	})

	t.Run("goal budget amounts nil goal", func(t *testing.T) {
		runGraphQLCase(t, "Common_SavingsGoalBudgetAmounts", map[string]any{"goalId": "goal-1", "startMonth": "2026-05-01", "endMonth": "2026-05-31"}, `{"savingsGoal":null}`, func(s *Service) error {
			got, err := s.GetGoalBudgetAmounts(context.Background(), "goal-1", "2026-05-01", "2026-05-31")
			mustNoErr(t, err)
			mustLen(t, got, 0)
			return nil
		})
	})

	t.Run("set goal budget amount", func(t *testing.T) {
		runGraphQLCase(t, "Common_SetSavingsGoalBudgetAmount", map[string]any{"input": map[string]any{"savingsGoalId": "goal-1", "month": "2026-06-01", "amount": 250.0, "applyToFuture": true, "accountId": "acc-1"}}, `{"setSavingsGoalBudgetAmount":{"success":true,"errors":null}}`, func(s *Service) error {
			return s.SetGoalBudgetAmount(context.Background(), "goal-1", "2026-06-01", 250, true, "acc-1")
		})
	})

	t.Run("set goal budget amount error", func(t *testing.T) {
		runGraphQLCase(t, "Common_SetSavingsGoalBudgetAmount", map[string]any{"input": map[string]any{"savingsGoalId": "goal-1", "month": "2026-06-01", "amount": 250.0, "applyToFuture": false}}, `{"setSavingsGoalBudgetAmount":{"success":false,"errors":[{"message":"bad"}]}}`, func(s *Service) error {
			hasErr(t, s.SetGoalBudgetAmount(context.Background(), "goal-1", "2026-06-01", 250, false, ""))
			return nil
		})
	})

	t.Run("set goal budget amount not successful", func(t *testing.T) {
		runGraphQLCase(t, "Common_SetSavingsGoalBudgetAmount", map[string]any{"input": map[string]any{"savingsGoalId": "goal-1", "month": "2026-06-01", "amount": 250.0, "applyToFuture": false}}, `{"setSavingsGoalBudgetAmount":{"success":false,"errors":null}}`, func(s *Service) error {
			hasErr(t, s.SetGoalBudgetAmount(context.Background(), "goal-1", "2026-06-01", 250, false, ""))
			return nil
		})
	})

	t.Run("goal client errors", func(t *testing.T) {
		svc := NewService(&mockClient{token: "t", handler: boomHandler})
		_, err := svc.GetGoal(context.Background(), "x")
		hasErr(t, err)
		_, err = svc.CreateGoal(context.Background(), &GoalInput{Name: "x"})
		hasErr(t, err)
		_, err = svc.UpdateGoal(context.Background(), "x", &GoalUpdate{})
		hasErr(t, err)
		hasErr(t, svc.DeleteGoal(context.Background(), "x"))
		_, err = svc.ArchiveGoal(context.Background(), "x")
		hasErr(t, err)
		_, err = svc.RestoreGoal(context.Background(), "x")
		hasErr(t, err)
		hasErr(t, svc.UpdateGoalPriorities(context.Background(), []string{"x"}))
		_, err = svc.LinkGoalAccount(context.Background(), "x", "a", nil, true)
		hasErr(t, err)
		_, err = svc.UnlinkGoalAccount(context.Background(), "x", "a")
		hasErr(t, err)
		_, err = svc.ListGoalEvents(context.Background(), "x")
		hasErr(t, err)
		_, err = svc.ContributeToGoal(context.Background(), "x", "a", 1, nil, nil)
		hasErr(t, err)
		_, err = svc.WithdrawFromGoal(context.Background(), "x", "a", 1, nil, nil)
		hasErr(t, err)
		_, err = svc.UpdateGoalEvent(context.Background(), "x", nil, nil)
		hasErr(t, err)
		hasErr(t, svc.DeleteGoalEvent(context.Background(), "x"))
		_, err = svc.GetGoalBudgetAmounts(context.Background(), "x", "2026-05-01", "2026-05-31")
		hasErr(t, err)
		hasErr(t, svc.SetGoalBudgetAmount(context.Background(), "x", "2026-06-01", 1, false, ""))
	})
}

func TestServiceParityGapPaths(t *testing.T) {
	t.Run("debt paydown", func(t *testing.T) {
		runGraphQLCase(t, "GetDebtPaydown", map[string]any{"input": map[string]any{"debtPaydownMethod": "planned"}}, `{"debtAccounts":[{"id":"a1","displayName":"Card","displayBalance":100,"apr":19.99,"interestRate":null,"minimumPayment":25,"plannedPayment":50,"excludeFromDebtPaydown":false},{"id":"a2","displayName":"Old","displayBalance":10,"apr":null,"interestRate":null,"minimumPayment":null,"plannedPayment":null,"excludeFromDebtPaydown":true}],"debtPaydownPlan":{"currentDebtPrincipal":100,"projectedInterest":5,"projectedTotal":105,"debtFreeDate":"2027-01-01","adjustedDebtFreeDate":"","adjustedProjectedInterest":0,"adjustedProjectedTotal":0,"debtAccountProjections":[{"account":{"id":"a1","displayName":"Card"},"principal":100,"projectedInterest":5,"projectedTotal":105,"debtFreeDate":"2027-01-01"}]}}`, func(s *Service) error {
			got, err := s.GetDebtPaydown(context.Background(), "")
			mustNoErr(t, err)
			eq(t, "planned", got.Method)
			mustLen(t, got.IncludedAccounts, 1)
			mustLen(t, got.ExcludedAccounts, 1)
			mustLen(t, got.Projections, 1)
			eq(t, 100.0, got.CurrentPrincipal)
			return nil
		})
	})

	t.Run("reorder rule", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "GetTransactionRules":
					return client.respond(result, `{"transactionRules":[{"id":"r1","order":0},{"id":"r2","order":1}]}`)
				case "Web_UpdateRuleOrderMutation":
					expectVars(t, req.Variables, map[string]any{"id": "r2", "order": 0})
					return client.respond(result, `{"updateTransactionRuleOrderV2":{"transactionRules":[{"id":"r2","order":0},{"id":"r1","order":1}]}}`)
				default:
					t.Fatalf("unexpected operation %q", req.OperationName)
					return nil
				}
			},
		}
		got, err := NewService(client).ReorderRule(context.Background(), "r2", 0)
		mustNoErr(t, err)
		eq(t, 1, got.MovedFrom)
		eq(t, 0, got.MovedTo)
		mustLen(t, got.Order, 2)
	})

	t.Run("reorder rule missing", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				return client.respond(result, `{"transactionRules":[{"id":"r1","order":0}]}`)
			},
		}
		_, err := NewService(client).ReorderRule(context.Background(), "nope", 0)
		if err == nil {
			t.Fatal("ReorderRule() error = nil, want not found")
		}
	})

	t.Run("connection health", func(t *testing.T) {
		runGraphQLCase(t, "GetCredentialSyncHealth", nil, `{"credentials":[{"id":"c1","updateRequired":true,"disconnectedFromDataProviderAt":"","syncDisabledAt":"","syncDisabledReason":"","dataProvider":"MX","displayLastUpdatedAt":"2020-01-01T00:00:00Z","institution":{"id":"i1","name":"Bank","url":"https://bank.example"},"accounts":[{"id":"a1","displayName":"Checking"}]},{"id":"c2","updateRequired":false,"disconnectedFromDataProviderAt":"","syncDisabledAt":"","syncDisabledReason":"","dataProvider":"plaid","displayLastUpdatedAt":"","institution":{"id":"i2","name":"CU","url":""},"accounts":[]}]}`, func(s *Service) error {
			got, err := s.GetConnectionHealth(context.Background(), 3)
			mustNoErr(t, err)
			eq(t, 2, got.ConnectionCount)
			eq(t, 1, got.NeedsAttentionCount)
			mustLen(t, got.NeedsAttention, 1)
			eq(t, "c1", got.NeedsAttention[0].CredentialID)
			return nil
		})
	})

	t.Run("whoami", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				switch req.OperationName {
				case "GetWhoAmI":
					return client.respond(result, `{"me":{"id":"u1","name":"Pat","email":"pat@example.com","timezone":"America/Vancouver","hasPassword":true,"externalAuthProviderNames":[]},"subscription":{"id":"s1","entitlements":["premium"],"hasPremiumEntitlement":true,"isOnFreeTrial":false,"billingPeriod":"yearly","currentPeriodEndsAt":"","trialEndsAt":"","willCancelAtPeriodEnd":false,"paymentSource":"","nextPaymentAmount":0}}`)
				case "ProbeBusinessEntities":
					return client.respond(result, `{"businessEntities":[{"id":"b1","name":"Biz","color":"red"}]}`)
				default:
					t.Fatalf("unexpected operation %q", req.OperationName)
					return nil
				}
			},
		}
		got, err := NewService(client).GetWhoAmI(context.Background())
		mustNoErr(t, err)
		eq(t, "pat@example.com", got.User.Email)
		eq(t, true, got.Subscription.HasPremium)
		eq(t, true, got.Capabilities.BusinessEntitiesAvailable)
		mustLen(t, got.Capabilities.BusinessEntities, 1)
	})

	t.Run("whoami probe failure", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "ProbeBusinessEntities" {
					return errors.New("boom")
				}
				return client.respond(result, `{"me":{"id":"u1","name":"Pat","email":"pat@example.com"},"subscription":null}`)
			},
		}
		got, err := NewService(client).GetWhoAmI(context.Background())
		mustNoErr(t, err)
		eq(t, false, got.Capabilities.BusinessEntitiesAvailable)
		mustLen(t, got.Capabilities.BusinessEntities, 0)
	})

	t.Run("review stream", func(t *testing.T) {
		runGraphQLCase(t, "Web_ReviewStream", map[string]any{"input": map[string]any{"streamId": "s1", "reviewStatus": "approved"}}, `{"reviewRecurringStream":{"stream":{"id":"s1","reviewStatus":"approved"},"errors":null}}`, func(s *Service) error {
			got, err := s.ReviewRecurringStream(context.Background(), "s1", "approved")
			mustNoErr(t, err)
			eq(t, "approved", got.ReviewStatus)
			return nil
		})
	})

	t.Run("review stream not found", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Web_ReviewStream")
				return client.respond(result, `{"reviewRecurringStream":{"stream":null,"errors":{"message":"Stream not found","code":null}}}`)
			},
		}
		_, err := NewService(client).ReviewRecurringStream(context.Background(), "nope", "approved")
		if err == nil || !strings.Contains(err.Error(), "Stream not found") {
			t.Fatalf("ReviewRecurringStream() error = %v, want 'Stream not found'", err)
		}
	})

	t.Run("goal contributions", func(t *testing.T) {
		runGraphQLCase(t, "GetSavingsGoalContributions", map[string]any{"id": "g1", "startMonth": "2026-09-01", "endMonth": "2026-09-30"}, `{"savingsGoal":{"id":"g1","name":"Rainy","monthlyBudgetAmounts":[{"month":"2026-09-01","totalPlannedAmount":50,"totalActualAmount":10,"totalRemainingAmount":40,"accountBreakdown":[{"account":{"id":"a1","displayName":"Checking"},"plannedAmount":50,"actualAmount":10,"remainingAmount":40}]}]}}`, func(s *Service) error {
			got, err := s.GetGoalContributions(context.Background(), "g1", "2026-09-01", "2026-09-30")
			mustNoErr(t, err)
			eq(t, "Rainy", got.GoalName)
			mustLen(t, got.Months, 1)
			mustLen(t, got.Months[0].Accounts, 1)
			eq(t, 50.0, got.Months[0].TotalPlanned)
			return nil
		})
	})

	t.Run("goal contributions missing", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "GetSavingsGoalContributions")
				return client.respond(result, `{"savingsGoal":null}`)
			},
		}
		_, err := NewService(client).GetGoalContributions(context.Background(), "nope", "2026-09-01", "2026-09-30")
		if err == nil {
			t.Fatal("GetGoalContributions() error = nil, want not found")
		}
	})

	t.Run("set goal contribution", func(t *testing.T) {
		runGraphQLCase(t, "Common_UpdateSavingsGoal", map[string]any{"input": map[string]any{"id": "g1", "accountBudgetAmounts": []any{map[string]any{"accountId": "a1", "amount": 50.0}}}}, `{"updateSavingsGoal":{"savingsGoal":{"id":"g1","type":"emergency_fund","name":"Rainy","status":"","progress":0,"currentBalance":0,"targetDate":"","targetAmount":100,"plannedMonthlyContribution":0,"currentMonthPlannedContributionAmount":0,"spendingTotal":0,"netContribution":0,"estimatedMonthsUntilCompletion":0,"forecastedCompletionDate":"","isSinkingFund":false,"priority":0},"errors":[]}}`, func(s *Service) error {
			got, err := s.SetGoalContribution(context.Background(), "g1", "a1", 50)
			mustNoErr(t, err)
			eq(t, "g1", got.ID)
			return nil
		})
	})
}

func TestServiceParityGapErrorPaths(t *testing.T) {
	t.Run("debt paydown transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetDebtPaydown", map[string]any{"input": map[string]any{"debtPaydownMethod": "planned"}}, func(s *Service) error {
			_, err := s.GetDebtPaydown(context.Background(), "")
			return err
		})
	})

	t.Run("debt paydown empty", func(t *testing.T) {
		runGraphQLCase(t, "GetDebtPaydown", map[string]any{"input": map[string]any{"debtPaydownMethod": "avalanche"}}, `{"debtAccounts":[],"debtPaydownPlan":null}`, func(s *Service) error {
			got, err := s.GetDebtPaydown(context.Background(), "avalanche")
			mustNoErr(t, err)
			mustLen(t, got.IncludedAccounts, 0)
			mustLen(t, got.ExcludedAccounts, 0)
			mustLen(t, got.Projections, 0)
			return nil
		})
	})

	t.Run("reorder rule list failure", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetTransactionRules", nil, func(s *Service) error {
			_, err := s.ReorderRule(context.Background(), "r1", 0)
			return err
		})
	})

	t.Run("reorder rule mutation failure", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "GetTransactionRules" {
					return client.respond(result, `{"transactionRules":[{"id":"r1","order":0}]}`)
				}
				assertReq(t, req, "Web_UpdateRuleOrderMutation")
				return errors.New("boom")
			},
		}
		_, err := NewService(client).ReorderRule(context.Background(), "r1", 0)
		if err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("ReorderRule() error = %v, want boom", err)
		}
	})

	t.Run("reorder rule empty order", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				if req.OperationName == "GetTransactionRules" {
					return client.respond(result, `{"transactionRules":[{"id":"r1","order":0}]}`)
				}
				return client.respond(result, `{"updateTransactionRuleOrderV2":{"transactionRules":[]}}`)
			},
		}
		got, err := NewService(client).ReorderRule(context.Background(), "r1", 0)
		mustNoErr(t, err)
		eq(t, 0, got.MovedTo)
		mustLen(t, got.Order, 0)
	})

	t.Run("connection health transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetCredentialSyncHealth", nil, func(s *Service) error {
			_, err := s.GetConnectionHealth(context.Background(), 3)
			return err
		})
	})

	t.Run("connection health branches", func(t *testing.T) {
		now := time.Now().Format(time.RFC3339)
		runGraphQLCase(t, "GetCredentialSyncHealth", nil, `{"credentials":[
			{"id":"c1","updateRequired":false,"disconnectedFromDataProviderAt":"2026-01-01T00:00:00Z","syncDisabledAt":"2026-02-01T00:00:00Z","syncDisabledReason":"user paused","dataProvider":"MX","displayLastUpdatedAt":"`+now+`","institution":{"id":"i1","name":"Clean-ish","url":""},"accounts":[]},
			{"id":"c2","updateRequired":false,"disconnectedFromDataProviderAt":"","syncDisabledAt":"2026-02-01T00:00:00Z","syncDisabledReason":"","dataProvider":"plaid","displayLastUpdatedAt":"not-a-date","institution":{"id":"i2","name":"Fresh","url":""},"accounts":[]},
			{"id":"c3","updateRequired":false,"disconnectedFromDataProviderAt":"","syncDisabledAt":"","syncDisabledReason":"","dataProvider":"plaid","displayLastUpdatedAt":"","institution":{"id":"i3","name":"Fine","url":""},"accounts":[]}
		]}`, func(s *Service) error {
			got, err := s.GetConnectionHealth(context.Background(), 3)
			mustNoErr(t, err)
			eq(t, 3, got.ConnectionCount)
			eq(t, 2, got.NeedsAttentionCount)
			mustLen(t, got.Connections[2].Reasons, 0)
			mustLen(t, got.Connections[2].Accounts, 0)
			return nil
		})
	})

	t.Run("connection health empty", func(t *testing.T) {
		runGraphQLCase(t, "GetCredentialSyncHealth", nil, `{"credentials":[]}`, func(s *Service) error {
			got, err := s.GetConnectionHealth(context.Background(), 3)
			mustNoErr(t, err)
			mustLen(t, got.Connections, 0)
			mustLen(t, got.NeedsAttention, 0)
			return nil
		})
	})

	t.Run("whoami transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetWhoAmI", nil, func(s *Service) error {
			_, err := s.GetWhoAmI(context.Background())
			return err
		})
	})

	t.Run("whoami missing me", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "GetWhoAmI")
				return client.respond(result, `{"me":null,"subscription":null}`)
			},
		}
		_, err := NewService(client).GetWhoAmI(context.Background())
		if err == nil {
			t.Fatal("GetWhoAmI() error = nil, want missing me")
		}
	})

	t.Run("review stream transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Web_ReviewStream", map[string]any{"input": map[string]any{"streamId": "s1", "reviewStatus": "approved"}}, func(s *Service) error {
			_, err := s.ReviewRecurringStream(context.Background(), "s1", "approved")
			return err
		})
	})

	t.Run("review stream missing stream", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Web_ReviewStream")
				return client.respond(result, `{"reviewRecurringStream":{"stream":null,"errors":null}}`)
			},
		}
		_, err := NewService(client).ReviewRecurringStream(context.Background(), "s1", "approved")
		if err == nil {
			t.Fatal("ReviewRecurringStream() error = nil, want missing stream")
		}
	})

	t.Run("goal contributions transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetSavingsGoalContributions", map[string]any{"id": "g1", "startMonth": "2026-09-01", "endMonth": "2026-09-30"}, func(s *Service) error {
			_, err := s.GetGoalContributions(context.Background(), "g1", "2026-09-01", "2026-09-30")
			return err
		})
	})

	t.Run("goal contributions empty", func(t *testing.T) {
		runGraphQLCase(t, "GetSavingsGoalContributions", map[string]any{"id": "g1", "startMonth": "2026-09-01", "endMonth": "2026-09-30"}, `{"savingsGoal":{"id":"g1","name":"Rainy","monthlyBudgetAmounts":[{"month":"2026-09-01","totalPlannedAmount":0,"totalActualAmount":0,"totalRemainingAmount":0,"accountBreakdown":[]}]}}`, func(s *Service) error {
			got, err := s.GetGoalContributions(context.Background(), "g1", "2026-09-01", "2026-09-30")
			mustNoErr(t, err)
			mustLen(t, got.Months, 1)
			mustLen(t, got.Months[0].Accounts, 0)
			return nil
		})
	})

	t.Run("goal contributions no months", func(t *testing.T) {
		runGraphQLCase(t, "GetSavingsGoalContributions", map[string]any{"id": "g1", "startMonth": "2026-09-01", "endMonth": "2026-09-30"}, `{"savingsGoal":{"id":"g1","name":"Rainy","monthlyBudgetAmounts":[]}}`, func(s *Service) error {
			got, err := s.GetGoalContributions(context.Background(), "g1", "2026-09-01", "2026-09-30")
			mustNoErr(t, err)
			mustLen(t, got.Months, 0)
			return nil
		})
	})

	t.Run("set goal contribution transport error", func(t *testing.T) {
		runGraphQLErrorCase(t, "Common_UpdateSavingsGoal", map[string]any{"input": map[string]any{"id": "g1", "accountBudgetAmounts": []any{map[string]any{"accountId": "a1", "amount": 50.0}}}}, func(s *Service) error {
			_, err := s.SetGoalContribution(context.Background(), "g1", "a1", 50)
			return err
		})
	})

	t.Run("set goal contribution api error", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_UpdateSavingsGoal")
				return client.respond(result, `{"updateSavingsGoal":{"savingsGoal":null,"errors":[{"message":"denied"}]}}`)
			},
		}
		_, err := NewService(client).SetGoalContribution(context.Background(), "g1", "a1", 50)
		if err == nil || !strings.Contains(err.Error(), "denied") {
			t.Fatalf("SetGoalContribution() error = %v, want 'denied'", err)
		}
	})

	t.Run("set goal contribution missing goal", func(t *testing.T) {
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result any) error {
				assertReq(t, req, "Common_UpdateSavingsGoal")
				return client.respond(result, `{"updateSavingsGoal":{"savingsGoal":null,"errors":[]}}`)
			},
		}
		_, err := NewService(client).SetGoalContribution(context.Background(), "g1", "a1", 50)
		if err == nil {
			t.Fatal("SetGoalContribution() error = nil, want missing savingsGoal")
		}
	})
}
